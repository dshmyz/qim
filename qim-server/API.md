# QIM API 文档

> QIM 即时通讯系统后端 API 参考文档
> 版本: 1.0.0
> 更新时间: 2026-04-16

---

## 目录

- [概述](#概述)
- [认证](#认证)
- [用户](#用户)
- [会话](#会话)
- [消息](#消息)
- [组织架构](#组织架构)
- [文件](#文件)
- [笔记](#笔记)
- [任务](#任务)
- [日历](#日历)
- [便签](#便签)
- [应用](#应用)
- [通知](#通知)
- [系统消息](#系统消息)
- [WebSocket](#websocket)

---

## 概述

### 基本信息

- **基础URL**: `http://localhost:8080`
- **API版本**: v1
- **认证方式**: Bearer Token (JWT)
- **数据格式**: JSON

### 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0表示成功 |
| message | string | 状态信息 |
| data | object | 响应数据 |

### 状态码

| 状态码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或认证失败 |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 认证

### 登录

用户登录系统，获取访问令牌。

```
POST /api/v1/auth/login
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| version | string | 否 | 客户端版本号 |

**请求示例**

```json
{
  "username": "admin",
  "password": "123456",
  "version": "1.0.0"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "nickname": "管理员",
      "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
      "signature": "这个人很懒，什么都没写",
      "phone": "13800138000",
      "email": "admin@qim.com",
      "two_factor_enabled": false
    }
  }
}
```

---

### 注册

创建新用户账号。

```
POST /api/v1/auth/register
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 (3-50字符) |
| password | string | 是 | 密码 (6-20字符) |
| nickname | string | 否 | 昵称 |

**请求示例**

```json
{
  "username": "zhangsan",
  "password": "123456",
  "nickname": "张三"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 2,
      "username": "zhangsan",
      "nickname": "张三",
      "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan"
    }
  }
}
```

---

### 刷新 Token

刷新访问令牌。

```
POST /api/v1/auth/refresh
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

### 登出

注销当前会话。

```
POST /api/v1/auth/logout
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "登出成功"
}
```

---

### 检查双因素认证状态

检查用户是否启用了双因素认证。

```
POST /api/v1/auth/check-2fa
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |

**响应示例**

```json
{
  "code": 0,
  "data": {
    "twoFactorEnabled": false
  }
}
```

---

### 双因素认证验证

验证双因素认证码。

```
POST /api/v1/auth/2fa/verify
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session | string | 是 | 会话ID |
| code | string | 是 | 6位验证码 |
| username | string | 是 | 用户名 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "nickname": "管理员"
    }
  }
}
```

---

### 重新发送验证码

重新发送双因素认证验证码。

```
POST /api/v1/auth/2fa/resend
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session | string | 是 | 会话ID |
| username | string | 是 | 用户名 |

**响应示例**

```json
{
  "code": 0,
  "message": "验证码已发送"
}
```

---

## 用户

### 获取当前用户

获取已登录用户的信息。

```
GET /api/v1/users/me
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "admin",
    "nickname": "管理员",
    "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
    "signature": "这个人很懒，什么都没写",
    "phone": "13800138000",
    "email": "admin@qim.com",
    "status": "online",
    "two_factor_enabled": false
  }
}
```

---

### 更新用户资料

更新当前用户的个人资料。

```
PUT /api/v1/users/me
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| nickname | string | 否 | 昵称 |
| avatar | string | 否 | 头像URL |
| signature | string | 否 | 个性签名 |
| phone | string | 否 | 手机号 |
| email | string | 否 | 邮箱 |
| two_factor_enabled | bool | 否 | 是否启用双因素认证 |

**请求示例**

```json
{
  "nickname": "新昵称",
  "signature": "新的个性签名",
  "avatar": "https://example.com/avatar.jpg"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "admin",
    "nickname": "新昵称",
    "avatar": "https://example.com/avatar.jpg",
    "signature": "新的个性签名",
    "phone": "13800138000",
    "email": "admin@qim.com",
    "status": "online"
  }
}
```

---

### 获取用户信息

获取指定用户的信息。

```
GET /api/v1/users/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "username": "zhangsan",
    "nickname": "张三",
    "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan",
    "signature": "专注工作",
    "phone": "13800138001",
    "email": "zhangsan@qim.com",
    "status": "offline"
  }
}
```

---

## 会话

### 获取会话列表

获取当前用户的所有会话列表。

```
GET /api/v1/conversations
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "type": "single",
      "name": "张三",
      "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan",
      "is_pinned": false,
      "ip": "192.168.1.100",
      "members": [
        {
          "id": 1,
          "user_id": 1,
          "role": "member",
          "unread_count": 0,
          "muted": false
        },
        {
          "id": 2,
          "user_id": 2,
          "role": "member",
          "unread_count": 3,
          "muted": false
        }
      ],
      "last_message": {
        "id": 100,
        "content": "你好",
        "created_at": "2026-04-16T10:30:00Z"
      }
    },
    {
      "id": 2,
      "type": "group",
      "name": "技术交流群",
      "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=group",
      "is_pinned": true,
      "members": [
        {
          "id": 3,
          "user_id": 1,
          "role": "owner",
          "unread_count": 5,
          "muted": false
        }
      ],
      "last_message": {
        "id": 99,
        "content": "今天讨论什么话题？",
        "created_at": "2026-04-16T09:15:00Z"
      }
    }
  ]
}
```

---

### 创建单聊会话

与指定用户创建单聊会话。

```
POST /api/v1/conversations/single
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | uint | 是 | 目标用户ID |

**请求示例**

```json
{
  "user_id": 2
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "type": "single",
    "name": "张三",
    "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan"
  }
}
```

---

### 创建群聊会话

创建新的群聊会话。

```
POST /api/v1/conversations/group
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 群聊名称 |
| avatar | string | 否 | 群聊头像 |
| member_ids | []uint | 是 | 成员ID列表 |

**请求示例**

```json
{
  "name": "技术交流群",
  "avatar": "https://example.com/group.jpg",
  "member_ids": [2, 3, 4]
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "type": "group",
    "name": "技术交流群",
    "avatar": "https://example.com/group.jpg"
  }
}
```

---

### 创建讨论组会话

创建讨论组会话（与群聊类似，所有成员平等）。

```
POST /api/v1/conversations/discussion
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 讨论组名称 |
| avatar | string | 否 | 讨论组头像 |
| member_ids | []uint | 是 | 成员ID列表 |

**请求示例**

```json
{
  "name": "产品讨论组",
  "member_ids": [2, 3]
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 3,
    "type": "discussion",
    "name": "产品讨论组"
  }
}
```

---

### 获取会话详情

获取指定会话的详细信息。

```
GET /api/v1/conversations/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "type": "group",
    "name": "技术交流群",
    "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=group",
    "creator_id": 1,
    "members": [
      {
        "id": 1,
        "user_id": 1,
        "role": "owner",
        "unread_count": 0,
        "muted": false,
        "user": {
          "id": 1,
          "username": "admin",
          "nickname": "管理员",
          "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=admin"
        }
      }
    ]
  }
}
```

---

### 会话置顶/取消置顶

设置或取消会话置顶。

```
PUT /api/v1/conversations/:id/pin
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| is_pinned | bool | 是 | 是否置顶 |

**请求示例**

```json
{
  "is_pinned": true
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 设置会话免打扰

设置或取消会话免打扰。

```
PUT /api/v1/conversations/:id/mute
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| muted | bool | 是 | 是否免打扰 |

**请求示例**

```json
{
  "muted": true
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 消息

### 获取历史消息

获取指定会话的历史消息。

```
GET /api/v1/conversations/:id/messages
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20，最大100 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 100,
      "conversation_id": 1,
      "sender_id": 2,
      "type": "text",
      "content": "你好",
      "file_name": null,
      "file_size": 0,
      "quoted_message_id": null,
      "is_recalled": false,
      "is_read": true,
      "created_at": "2026-04-16T10:30:00Z",
      "sender": {
        "id": 2,
        "username": "zhangsan",
        "nickname": "张三",
        "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan"
      },
      "quoted_message": null,
      "share_data": null
    },
    {
      "id": 101,
      "conversation_id": 1,
      "sender_id": 1,
      "type": "image",
      "content": "https://example.com/image.jpg",
      "file_name": null,
      "file_size": 123456,
      "quoted_message_id": 100,
      "is_recalled": false,
      "is_read": false,
      "created_at": "2026-04-16T10:31:00Z",
      "sender": {
        "id": 1,
        "username": "admin",
        "nickname": "管理员",
        "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=admin"
      },
      "quoted_message": {
        "id": 100,
        "sender_id": 2,
        "type": "text",
        "content": "你好",
        "sender": {
          "id": 2,
          "nickname": "张三"
        }
      },
      "share_data": null
    }
  ],
  "pagination": {
    "current_page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

### 发送消息

向指定会话发送消息。

```
POST /api/v1/conversations/:id/messages
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 消息类型: text/image/file/share/miniApp/news |
| content | string | 是 | 消息内容 |
| quoted_message_id | uint | 否 | 引用的消息ID |
| file_size | int64 | 否 | 文件大小（文件消息） |
| file_name | string | 否 | 文件名（文件消息） |
| share_data | object | 否 | 分享数据（分享消息） |

**请求示例 - 文本消息**

```json
{
  "type": "text",
  "content": "这是一条测试消息"
}
```

**请求示例 - 引用消息**

```json
{
  "type": "text",
  "content": "好的，收到！",
  "quoted_message_id": 100
}
```

**请求示例 - 分享消息**

```json
{
  "type": "share",
  "content": "{\"type\":\"note\",\"name\":\"学习笔记\"}",
  "share_data": {
    "type": "note",
    "name": "学习笔记",
    "id": 1
  }
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 102,
    "conversation_id": 1,
    "sender_id": 1,
    "type": "text",
    "content": "这是一条测试消息",
    "created_at": "2026-04-16T10:35:00Z"
  }
}
```

---

### 按条件搜索消息

根据条件搜索消息。

```
GET /api/v1/messages/search
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversation_id | int | 是 | 会话ID |
| type | string | 否 | 消息类型 |
| search | string | 否 | 搜索关键词 |
| start_date | string | 否 | 开始日期 (RFC3339格式) |
| end_date | string | 否 | 结束日期 (RFC3339格式) |
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认10，最大100 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "messages": [
      {
        "id": 50,
        "conversation_id": 1,
        "sender_id": 2,
        "type": "text",
        "content": "会议通知",
        "created_at": "2026-04-15T14:00:00Z",
        "sender": {
          "id": 2,
          "nickname": "张三"
        }
      }
    ],
    "total": 1
  }
}
```

---

### 撤回消息

撤回已发送的消息（仅发送者可在2分钟内撤回）。

```
PUT /api/v1/messages/:id/recall
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 删除消息

删除消息（标记为已删除）。

```
DELETE /api/v1/messages/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 组织架构

### 获取组织架构树

获取完整的组织架构树形结构。

```
GET /api/v1/organization/tree
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "总公司",
      "parent_id": null,
      "level": 1,
      "path": "1",
      "sort_order": 0,
      "subDepartments": [
        {
          "id": 2,
          "name": "技术部",
          "parent_id": 1,
          "level": 2,
          "path": "1/2",
          "sort_order": 0,
          "subDepartments": [
            {
              "id": 4,
              "name": "前端组",
              "parent_id": 2,
              "level": 3,
              "path": "1/2/4",
              "sort_order": 0,
              "subDepartments": [],
              "employees": [
                {
                  "id": 1,
                  "username": "admin",
                  "nickname": "管理员",
                  "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
                  "signature": "技术总监",
                  "status": "online"
                }
              ]
            }
          ],
          "employees": [
            {
              "id": 2,
              "username": "zhangsan",
              "nickname": "张三",
              "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan",
              "signature": "高级工程师",
              "status": "offline"
            }
          ]
        },
        {
          "id": 3,
          "name": "产品部",
          "parent_id": 1,
          "level": 2,
          "path": "1/3",
          "sort_order": 1,
          "subDepartments": [],
          "employees": []
        }
      ],
      "employees": []
    }
  ]
}
```

---

### 创建部门

创建新部门。

```
POST /api/v1/departments
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 部门名称 |
| parent_id | uint | 否 | 父部门ID |
| level | int | 是 | 部门层级 |
| path | string | 否 | 部门路径 |
| sort_order | int | 否 | 排序序号 |

**请求示例**

```json
{
  "name": "前端组",
  "parent_id": 2,
  "level": 3,
  "path": "1/2/4",
  "sort_order": 0
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 4,
    "name": "前端组",
    "parent_id": 2,
    "level": 3,
    "path": "1/2/4",
    "sort_order": 0
  }
}
```

---

### 创建用户

创建新用户。

```
POST /api/v1/users
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| nickname | string | 否 | 昵称 |
| avatar | string | 否 | 头像URL |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 5,
    "username": "wangwu",
    "nickname": "王五",
    "avatar": "https://api.dicebear.com/7.x/avataaars/svg?seed=wangwu",
    "status": "offline"
  }
}
```

---

### 关联用户到部门

将用户添加到指定部门。

```
POST /api/v1/departments/:id/users
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | uint | 是 | 用户ID |
| position | string | 否 | 职位 |
| is_primary | bool | 否 | 是否为主部门 |

**请求示例**

```json
{
  "user_id": 5,
  "position": "前端工程师",
  "is_primary": true
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "user_id": 5,
    "department_id": 4,
    "position": "前端工程师",
    "is_primary": true
  }
}
```

---

## 文件

### 上传文件

上传文件到服务器。

```
POST /api/v1/files/upload
Authorization: Bearer {token}
Content-Type: multipart/form-data
```

**表单参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 文件 |
| folder_id | uint | 否 | 目标文件夹ID |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "document.pdf",
    "original_name": "文档.pdf",
    "size": 1024000,
    "mime_type": "application/pdf",
    "url": "http://localhost:8080/uploads/20260416100000_1.pdf"
  }
}
```

---

### 获取文件列表

获取用户的文件列表。

```
GET /api/v1/files
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| folder_id | uint | 否 | 文件夹ID |
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "document.pdf",
      "original_name": "文档.pdf",
      "size": 1024000,
      "mime_type": "application/pdf",
      "url": "http://localhost:8080/uploads/20260416100000_1.pdf",
      "created_at": "2026-04-16T10:00:00Z"
    }
  ]
}
```

---

### 下载文件

下载指定文件。

```
GET /api/v1/files/:id/download
Authorization: Bearer {token}
```

**响应**: 文件流

---

### 删除文件

删除指定文件。

```
DELETE /api/v1/files/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 笔记

### 获取笔记列表

获取用户的所有笔记。

```
GET /api/v1/notes
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "title": "学习计划",
      "content": "# 本周学习计划\n\n- [x] 学习 Vue 3\n- [ ] 学习 TypeScript",
      "color": "yellow",
      "type": "note",
      "created_at": "2026-04-15T10:00:00Z",
      "updated_at": "2026-04-16T09:00:00Z"
    }
  ]
}
```

---

### 获取笔记详情

获取指定笔记的详细信息。

```
GET /api/v1/notes/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "user_id": 1,
    "title": "学习计划",
    "content": "# 本周学习计划\n\n- [x] 学习 Vue 3\n- [ ] 学习 TypeScript",
    "color": "yellow",
    "type": "note",
    "created_at": "2026-04-15T10:00:00Z",
    "updated_at": "2026-04-16T09:00:00Z"
  }
}
```

---

### 创建笔记

创建新笔记。

```
POST /api/v1/notes
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 笔记标题 |
| content | string | 是 | 笔记内容 (Markdown) |
| color | string | 否 | 颜色，默认yellow |

**请求示例**

```json
{
  "title": "新笔记",
  "content": "# 新笔记内容\n\n这是我的新笔记。",
  "color": "blue"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "title": "新笔记",
    "content": "# 新笔记内容\n\n这是我的新笔记。",
    "color": "blue",
    "created_at": "2026-04-16T11:00:00Z"
  }
}
```

---

### 更新笔记

更新指定笔记。

```
PUT /api/v1/notes/:id
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 否 | 笔记标题 |
| content | string | 否 | 笔记内容 |
| color | string | 否 | 颜色 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "title": "更新后的笔记",
    "content": "# 更新后的内容",
    "updated_at": "2026-04-16T12:00:00Z"
  }
}
```

---

### 删除笔记

删除指定笔记。

```
DELETE /api/v1/notes/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 任务

### 获取任务列表

获取用户的任务列表。

```
GET /api/v1/tasks
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "title": "完成项目文档",
      "description": "编写项目技术文档",
      "status": "todo",
      "priority": "high",
      "due_date": "2026-04-20",
      "created_at": "2026-04-15T10:00:00Z",
      "updated_at": "2026-04-15T10:00:00Z"
    }
  ]
}
```

---

### 创建任务

创建新任务。

```
POST /api/v1/tasks
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 任务标题 |
| description | string | 否 | 任务描述 |
| status | string | 否 | 状态: todo/in_progress/completed |
| priority | string | 否 | 优先级: low/medium/high |
| due_date | string | 否 | 截止日期 (YYYY-MM-DD格式) |

**请求示例**

```json
{
  "title": "完成项目文档",
  "description": "编写项目技术文档",
  "status": "todo",
  "priority": "high",
  "due_date": "2026-04-20"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "title": "完成项目文档",
    "description": "编写项目技术文档",
    "status": "todo",
    "priority": "high",
    "due_date": "2026-04-20",
    "created_at": "2026-04-16T11:00:00Z"
  }
}
```

---

### 更新任务状态

更新任务状态。

```
PUT /api/v1/tasks/:id/status
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | string | 是 | 状态: todo/in_progress/completed |

**请求示例**

```json
{
  "status": "completed"
}
```

---

### 删除任务

删除指定任务。

```
DELETE /api/v1/tasks/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 日历

### 获取日历事件列表

获取用户的日历事件。

```
GET /api/v1/events
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "title": "团队会议",
      "description": "讨论项目进度",
      "start": "2026-04-16T14:00:00Z",
      "end": "2026-04-16T15:00:00Z",
      "all_day": false,
      "reminder": 15,
      "created_at": "2026-04-15T10:00:00Z"
    }
  ]
}
```

---

### 创建日历事件

创建新日历事件。

```
POST /api/v1/events
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 事件标题 |
| description | string | 否 | 事件描述 |
| start | string | 是 | 开始时间 (ISO8601格式) |
| end | string | 是 | 结束时间 (ISO8601格式) |
| all_day | bool | 否 | 是否全天事件 |
| reminder | int | 否 | 提醒时间（分钟），0表示不提醒 |

**请求示例**

```json
{
  "title": "团队会议",
  "description": "讨论项目进度",
  "start": "2026-04-16T14:00:00Z",
  "end": "2026-04-16T15:00:00Z",
  "all_day": false,
  "reminder": 15
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "title": "团队会议",
    "description": "讨论项目进度",
    "start": "2026-04-16T14:00:00Z",
    "end": "2026-04-16T15:00:00Z",
    "all_day": false,
    "reminder": 15,
    "created_at": "2026-04-16T10:00:00Z"
  }
}
```

---

### 更新日历事件

更新指定日历事件。

```
PUT /api/v1/events/:id
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

同创建事件的参数。

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "title": "更新后的会议",
    "updated_at": "2026-04-16T11:00:00Z"
  }
}
```

---

### 删除日历事件

删除指定日历事件。

```
DELETE /api/v1/events/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 便签

### 获取便签列表

获取用户的所有便签。

```
GET /api/v1/stickies
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "content": "记得买牛奶",
      "color": "yellow",
      "created_at": "2026-04-16T09:00:00Z",
      "updated_at": "2026-04-16T09:00:00Z"
    }
  ]
}
```

---

### 创建便签

创建新便签。

```
POST /api/v1/stickies
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 便签内容 |
| color | string | 否 | 颜色: yellow/blue/red/green/purple |

**请求示例**

```json
{
  "content": "记得买牛奶",
  "color": "yellow"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "content": "记得买牛奶",
    "color": "yellow",
    "created_at": "2026-04-16T10:00:00Z"
  }
}
```

---

### 更新便签

更新指定便签。

```
PUT /api/v1/stickies/:id
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 否 | 便签内容 |
| color | string | 否 | 颜色 |

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 删除便签

删除指定便签。

```
DELETE /api/v1/stickies/:id
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 应用

### 获取应用列表

获取所有可用的应用列表。

```
GET /api/v1/apps
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "统计报表",
      "icon": "fas fa-chart-bar",
      "url": null,
      "category": "tools",
      "sort_order": 1,
      "is_active": true
    },
    {
      "id": 2,
      "name": "日历",
      "icon": "fas fa-calendar",
      "url": null,
      "category": "tools",
      "sort_order": 2,
      "is_active": true
    }
  ]
}
```

---

## 通知

### 获取通知列表

获取用户收到的通知列表。

```
GET /api/v1/notifications
Authorization: Bearer {token}
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "title": "新消息",
      "content": "张三发送了一条新消息",
      "type": "message",
      "read": false,
      "data": {
        "conversation_id": 1
      },
      "created_at": "2026-04-16T10:30:00Z"
    },
    {
      "id": 2,
      "user_id": 1,
      "title": "系统通知",
      "content": "您的账号已成功登录",
      "type": "system",
      "read": true,
      "data": null,
      "created_at": "2026-04-16T09:00:00Z"
    }
  ]
}
```

---

### 标记通知为已读

标记指定通知为已读。

```
PUT /api/v1/notifications/:id/read
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 清空所有通知

清空用户的所有通知。

```
DELETE /api/v1/notifications
Authorization: Bearer {token}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success"
}
```

---

## 系统消息

### 发布系统消息

发布系统消息（仅管理员）。

```
POST /api/v1/system-messages
Authorization: Bearer {token}
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 消息标题 |
| content | string | 是 | 消息内容 |
| target | string | 否 | 发送范围: all/group/user，默认all |
| group_id | uint | 否 | 目标群聊ID（target为group时） |
| user_id | uint | 否 | 目标用户ID（target为user时） |

**请求示例**

```json
{
  "title": "系统升级通知",
  "content": "系统将于今晚22:00进行升级维护",
  "target": "all"
}
```

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "系统升级通知",
    "content": "系统将于今晚22:00进行升级维护",
    "target": "all",
    "created_at": "2026-04-16T11:00:00Z"
  }
}
```

---

## WebSocket

### 连接 WebSocket

建立 WebSocket 连接。

```
GET /ws?token={token}
Upgrade: websocket
Connection: Upgrade
```

**连接示例**

```
ws://localhost:8080/ws?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

### 消息格式

所有 WebSocket 消息采用 JSON 格式：

```json
{
  "type": "message_type",
  "data": {},
  "request_id": "uuid" 
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 消息类型 |
| data | object | 消息数据 |
| request_id | string | 请求ID（可选） |

---

### 客户端发送消息

| 类型 | 说明 | 数据结构 |
|------|------|----------|
| heartbeat | 心跳检测 | `{}` |
| send_message | 发送消息 | `{conversation_id, type, content, quoted_message_id, file_size, file_name, share_data}` |
| read_message | 标记消息已读 | `{conversation_id, message_id}` |
| webrtc_offer | WebRTC呼叫 | `{target_user_id, offer}` |
| webrtc_answer | WebRTC应答 | `{target_user_id, answer}` |
| webrtc_ice_candidate | ICE候选 | `{target_user_id, candidate}` |

**发送消息示例**

```json
{
  "type": "send_message",
  "data": {
    "conversation_id": 1,
    "type": "text",
    "content": "你好"
  },
  "request_id": "msg-123456"
}
```

**心跳示例**

```json
{
  "type": "heartbeat",
  "data": {}
}
```

---

### 服务端推送消息

| 类型 | 说明 | 数据结构 |
|------|------|----------|
| new_message | 新消息 | Message对象 |
| message_recalled | 消息撤回 | `{message_id, conversation_id}` |
| message_deleted | 消息删除 | `{message_id, conversation_id}` |
| user_status | 用户状态变更 | `{user_id, status}` |
| ack | 消息确认 | `{request_id, message_id}` |
| error | 错误消息 | `{code, message}` |

**新消息推送示例**

```json
{
  "type": "new_message",
  "data": {
    "id": 102,
    "conversation_id": 1,
    "sender_id": 1,
    "type": "text",
    "content": "你好",
    "created_at": "2026-04-16T10:35:00Z",
    "sender": {
      "id": 1,
      "nickname": "管理员"
    }
  }
}
```

**消息确认示例**

```json
{
  "type": "ack",
  "data": {
    "request_id": "msg-123456",
    "message_id": 102
  }
}
```

**用户状态变更示例**

```json
{
  "type": "user_status",
  "data": {
    "user_id": 2,
    "status": "online"
  }
}
```

---

### 心跳机制

- 客户端每 30 秒发送一次心跳
- 服务端 5 分钟未收到心跳则断开连接
- 心跳包同时更新用户在线状态

---

## 错误码详情

| 错误码 | HTTP状态码 | 说明 |
|--------|------------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未认证或Token无效 |
| 403 | 403 | 无权限访问 |
| 404 | 404 | 资源不存在 |
| 500 | 500 | 服务器内部错误 |

---

## 附录

### 消息类型说明

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| text | 文本消息 | 普通文本聊天 |
| image | 图片消息 | 图片分享 |
| file | 文件消息 | 文件传输 |
| share | 分享消息 | 分享笔记、便签、文件等 |
| miniApp | 小程序消息 | 小程序卡片 |
| news | 资讯消息 | 新闻链接卡片 |

### 会话类型说明

| 类型 | 说明 |
|------|------|
| single | 单聊 |
| group | 群聊 |
| discussion | 讨论组 |
| channel | 频道 |

### 任务状态说明

| 状态 | 说明 |
|------|------|
| todo | 待办 |
| in_progress | 进行中 |
| completed | 已完成 |

### 任务优先级说明

| 优先级 | 说明 |
|--------|------|
| low | 低 |
| medium | 中 |
| high | 高 |

### 便签颜色说明

| 颜色 | 说明 |
|------|------|
| yellow | 黄色 |
| blue | 蓝色 |
| red | 红色 |
| green | 绿色 |
| purple | 紫色 |
