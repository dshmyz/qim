package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// errBoundary 表示部门已到达同级列表边界，无法继续移动。
var errBoundary = errors.New("boundary reached")

func GetOrganizationTree(c *gin.Context) {
	db := database.GetDB()

	var departments []model.Department
	if err := db.Where("parent_id IS NULL").Order("sort_order ASC, id ASC").Find(&departments).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	for i := range departments {
		loadDepartmentChildren(&departments[i], db)
	}

	var unassignedUsers []model.User
	db.Where("type = ? AND id NOT IN (SELECT user_id FROM department_employees)", "user").
		Find(&unassignedUsers)

	if len(unassignedUsers) > 0 {
		departments = append(departments, model.Department{
			Name:           "非标准用户",
			Employees:      unassignedUsers,
			SubDepartments: []model.Department{},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"departments": departmentsToFrontend(departments),
		},
	})
}

func CreateDepartment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		ParentID    *uint  `json:"parentId"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	level := 1
	path := "/"
	if req.ParentID != nil {
		var parent model.Department
		if err := db.First(&parent, *req.ParentID).Error; err != nil {
			response.NotFound(c, "上级部门不存在")
			return
		}
		level = parent.Level + 1
		path = parent.Path + "/" + path
	}

	department := model.Department{
		Name:      req.Name,
		ParentID:  req.ParentID,
		Level:     level,
		Path:      path,
		SortOrder: nextSortOrder(db, req.ParentID),
	}

	if err := db.Create(&department).Error; err != nil {
		response.InternalServerError(c, "创建部门失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departmentToFrontend(department),
	})
}

// UpdateDepartment 更新部门信息
func UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的部门ID")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var department model.Department
	if err := db.First(&department, uint(id)).Error; err != nil {
		response.NotFound(c, "部门不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Code != "" {
		updates["code"] = req.Code
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": departmentToFrontend(department)})
		return
	}

	if err := db.Model(&department).Updates(updates).Error; err != nil {
		response.InternalServerError(c, "更新部门失败")
		return
	}

	// 重新查询返回最新数据
	db.First(&department, uint(id))
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": departmentToFrontend(department)})
}

func AddUserToDepartment(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		DepartmentID uint   `json:"department_id" binding:"required"`
		Position     string `json:"position"`
		IsPrimary    bool   `json:"is_primary"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var user model.User
	if err := db.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var department model.Department
	if err := db.First(&department, req.DepartmentID).Error; err != nil {
		response.NotFound(c, "部门不存在")
		return
	}

	departmentEmployee := model.DepartmentEmployee{
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		Position:     req.Position,
		IsPrimary:    req.IsPrimary,
	}

	if err := db.Create(&departmentEmployee).Error; err != nil {
		response.InternalServerError(c, "关联用户和部门失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departmentEmployee,
	})
}

func loadDepartmentChildren(dept *model.Department, db *gorm.DB) {
	db.Where("parent_id = ?", dept.ID).Order("sort_order ASC, id ASC").Find(&dept.SubDepartments)
	for i := range dept.SubDepartments {
		loadDepartmentChildren(&dept.SubDepartments[i], db)
	}

	var deptEmps []model.DepartmentEmployee
	db.Where("department_id = ?", dept.ID).Preload("User").Find(&deptEmps)
	for _, de := range deptEmps {
		dept.Employees = append(dept.Employees, de.User)
	}
}

// DeleteDepartment 删除部门
func DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的部门ID")
		return
	}

	db := database.GetDB()

	var department model.Department
	if err := db.First(&department, uint(id)).Error; err != nil {
		response.NotFound(c, "部门不存在")
		return
	}

	// 检查是否有子部门
	var childCount int64
	db.Model(&model.Department{}).Where("parent_id = ?", department.ID).Count(&childCount)
	if childCount > 0 {
		response.BadRequest(c, "请先删除子部门")
		return
	}

	// 删除部门员工关联
	db.Where("department_id = ?", department.ID).Delete(&model.DepartmentEmployee{})

	// 删除部门
	if err := db.Delete(&department).Error; err != nil {
		response.InternalServerError(c, "删除部门失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetDepartmentEmployees 获取部门员工列表
func GetDepartmentEmployees(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的部门ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := database.GetDB()

	var total int64
	db.Model(&model.DepartmentEmployee{}).Where("department_id = ?", uint(id)).Count(&total)

	var deptEmployees []model.DepartmentEmployee
	offset := (page - 1) * pageSize
	if err := db.Where("department_id = ?", uint(id)).Preload("User").Offset(offset).Limit(pageSize).Find(&deptEmployees).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	type EmployeeInfo struct {
		ID         uint   `json:"id"`
		Username   string `json:"username"`
		Nickname   string `json:"nickname"`
		Email      string `json:"email"`
		Position   string `json:"position"`
		Department string `json:"department"`
	}

	employees := make([]EmployeeInfo, 0, len(deptEmployees))
	for _, de := range deptEmployees {
		employees = append(employees, EmployeeInfo{
			ID:       de.User.ID,
			Username: de.User.Username,
			Nickname: de.User.Nickname,
			Email:    de.User.Email,
			Position: de.Position,
		})
	}

	var department model.Department
	db.First(&department, uint(id))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  employees,
			"total": total,
		},
	})
}

// RemoveEmployeeFromDepartment 从部门移除员工
func RemoveEmployeeFromDepartment(c *gin.Context) {
	deptIDStr := c.Param("id")
	userIDStr := c.Param("user_id")

	deptID, err := strconv.ParseUint(deptIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的部门ID")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	db := database.GetDB()

	result := db.Where("department_id = ? AND user_id = ?", uint(deptID), uint(userID)).Delete(&model.DepartmentEmployee{})
	if result.Error != nil || result.RowsAffected == 0 {
		response.BadRequest(c, "移除失败或员工不在该部门")
		return
	}

	response.SuccessWithMessage(c, "移出成功", nil)
}

// siblingScope 按父部门约束查询范围，处理 parent_id 为 NULL 的根部门。
func siblingScope(db *gorm.DB, parentID *uint) *gorm.DB {
	if parentID == nil {
		return db.Where("parent_id IS NULL")
	}
	return db.Where("parent_id = ?", *parentID)
}

// nextSortOrder 返回同级部门末尾的排序值。
func nextSortOrder(db *gorm.DB, parentID *uint) int {
	var maxOrder *int
	siblingScope(db.Model(&model.Department{}), parentID).
		Select("MAX(sort_order)").Scan(&maxOrder)
	if maxOrder == nil {
		return 0
	}
	return *maxOrder + 1
}

// MoveDepartment 上移/下移部门，交换与同级相邻部门的排序值。
func MoveDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的部门ID")
		return
	}

	var req struct {
		Direction string `json:"direction" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if req.Direction != "up" && req.Direction != "down" {
		response.BadRequest(c, "方向参数无效")
		return
	}

	db := database.GetDB()

	var target model.Department
	if err := db.First(&target, uint(id)).Error; err != nil {
		response.NotFound(c, "部门不存在")
		return
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// 查出所有同级部门并按当前顺序规范化 sort_order（历史数据可能全为 0）
		var siblings []model.Department
		if err := siblingScope(tx, target.ParentID).
			Order("sort_order ASC, id ASC").Find(&siblings).Error; err != nil {
			return err
		}

		idx := -1
		for i := range siblings {
			if siblings[i].ID == target.ID {
				idx = i
			}
			if siblings[i].SortOrder != i {
				if err := tx.Model(&model.Department{}).
					Where("id = ?", siblings[i].ID).
					Update("sort_order", i).Error; err != nil {
					return err
				}
				siblings[i].SortOrder = i
			}
		}

		swapWith := idx - 1
		if req.Direction == "down" {
			swapWith = idx + 1
		}
		if idx == -1 || swapWith < 0 || swapWith >= len(siblings) {
			return errBoundary
		}

		a, b := siblings[idx], siblings[swapWith]
		if err := tx.Model(&model.Department{}).Where("id = ?", a.ID).
			Update("sort_order", b.SortOrder).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Department{}).Where("id = ?", b.ID).
			Update("sort_order", a.SortOrder).Error; err != nil {
			return err
		}
		return nil
	})

	if err == errBoundary {
		response.BadRequest(c, "已到达边界，无法移动")
		return
	}
	if err != nil {
		response.InternalServerError(c, "移动部门失败")
		return
	}

	response.SuccessWithMessage(c, "移动成功", nil)
}
