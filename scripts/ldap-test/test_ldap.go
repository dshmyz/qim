package main

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("  LDAP 连接测试")
	fmt.Println("==========================================")
	fmt.Println()

	addr := "localhost:389"
	fmt.Printf("连接到 %s ...\n", addr)

	conn, err := ldap.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	fmt.Println("连接成功!")
	fmt.Println()

	bindDN := "cn=admin,dc=qim,dc=local"
	bindPassword := "admin123"
	fmt.Printf("管理员绑定: %s\n", bindDN)

	err = conn.Bind(bindDN, bindPassword)
	if err != nil {
		log.Fatalf("管理员绑定失败: %v", err)
	}

	fmt.Println("管理员绑定成功!")
	fmt.Println()

	baseDN := "ou=users,dc=qim,dc=local"
	filter := "(uid=testuser1)"
	fmt.Printf("搜索用户: %s\n", filter)
	fmt.Printf("Base DN: %s\n", baseDN)
	fmt.Println()

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{"dn", "uid", "cn", "mail"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Fatalf("搜索失败: %v", err)
	}

	if len(sr.Entries) == 0 {
		log.Fatal("未找到用户")
	}

	fmt.Printf("找到 %d 个用户:\n", len(sr.Entries))
	for _, entry := range sr.Entries {
		fmt.Printf("  DN: %s\n", entry.DN)
		fmt.Printf("  UID: %s\n", entry.GetAttributeValue("uid"))
		fmt.Printf("  CN: %s\n", entry.GetAttributeValue("cn"))
		fmt.Printf("  Email: %s\n", entry.GetAttributeValue("mail"))
	}
	fmt.Println()

	userDN := sr.Entries[0].DN
	userPassword := "123456"
	fmt.Printf("用户绑定验证: %s\n", userDN)
	fmt.Printf("密码: %s\n", userPassword)
	fmt.Println()

	err = conn.Bind(userDN, userPassword)
	if err != nil {
		log.Fatalf("用户绑定失败(密码错误): %v", err)
	}

	fmt.Println("用户绑定成功! 密码正确!")
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  LDAP 测试完成!")
	fmt.Println("==========================================")
}
