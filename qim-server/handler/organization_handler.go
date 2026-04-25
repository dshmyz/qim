package handler

import (
	"net/http"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetOrganizationTree(c *gin.Context) {
	db := database.GetDB()

	var departments []model.Department
	if err := db.Where("parent_id IS NULL").Find(&departments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	for i := range departments {
		loadDepartmentChildren(&departments[i], db)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departments,
	})
}

func CreateDepartment(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		ParentID  *uint  `json:"parent_id"`
		Level     int    `json:"level" binding:"required"`
		Path      string `json:"path"`
		SortOrder int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	department := model.Department{
		Name:      req.Name,
		ParentID:  req.ParentID,
		Level:     req.Level,
		Path:      req.Path,
		SortOrder: req.SortOrder,
	}

	if err := db.Create(&department).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建部门失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": department,
	})
}

func AddUserToDepartment(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		DepartmentID uint   `json:"department_id" binding:"required"`
		Position     string `json:"position"`
		IsPrimary    bool   `json:"is_primary"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var user model.User
	if err := db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	var department model.Department
	if err := db.First(&department, req.DepartmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部门不存在"})
		return
	}

	departmentEmployee := model.DepartmentEmployee{
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		Position:     req.Position,
		IsPrimary:    req.IsPrimary,
	}

	if err := db.Create(&departmentEmployee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "关联用户和部门失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": departmentEmployee,
	})
}

func loadDepartmentChildren(dept *model.Department, db *gorm.DB) {
	db.Where("parent_id = ?", dept.ID).Find(&dept.SubDepartments)
	for i := range dept.SubDepartments {
		loadDepartmentChildren(&dept.SubDepartments[i], db)
	}

	var deptEmps []model.DepartmentEmployee
	db.Where("department_id = ?", dept.ID).Preload("User").Find(&deptEmps)
	for _, de := range deptEmps {
		dept.Employees = append(dept.Employees, de.User)
	}
}
