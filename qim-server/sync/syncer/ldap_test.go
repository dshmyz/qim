package syncer

import (
	"testing"

	"github.com/go-ldap/ldap/v3"
)

func TestLDAPUserDepartmentPrefersParentDN(t *testing.T) {
	s := &LDAPSyncer{config: &LDAPConfig{BaseDN: "ou=公司,dc=example,dc=com"}}
	dnToUUID := map[string]string{
		"ou=研发部,ou=公司,dc=example,dc=com": "dept-entry-uuid",
	}
	entry := ldap.NewEntry("uid=zhangsan,ou=研发部,ou=公司,dc=example,dc=com", map[string][]string{
		"departmentNumber": {"legacy-dept-number"},
	})

	got := s.resolveUserDepartmentID(entry, dnToUUID)

	if got != "dept-entry-uuid" {
		t.Fatalf("expected parent DN department ID, got %q", got)
	}
}

func TestLDAPUserDepartmentFallsBackToDepartmentNumber(t *testing.T) {
	s := &LDAPSyncer{config: &LDAPConfig{BaseDN: "ou=公司,dc=example,dc=com"}}
	entry := ldap.NewEntry("uid=zhangsan,ou=未知部,ou=公司,dc=example,dc=com", map[string][]string{
		"departmentNumber": {"dept-number"},
	})

	got := s.resolveUserDepartmentID(entry, map[string]string{})

	if got != "dept-number" {
		t.Fatalf("expected departmentNumber fallback, got %q", got)
	}
}

func TestLDAPUserDepartmentSkipsExcludedBaseDNParent(t *testing.T) {
	includeBase := false
	s := &LDAPSyncer{config: &LDAPConfig{
		BaseDN:        "ou=公司,dc=example,dc=com",
		IncludeBaseDN: &includeBase,
	}}
	entry := ldap.NewEntry("uid=zhangsan,ou=公司,dc=example,dc=com", nil)
	dnToUUID := map[string]string{
		"ou=公司,dc=example,dc=com": "company-entry-uuid",
	}

	got := s.resolveUserDepartmentID(entry, dnToUUID)

	if got != "" {
		t.Fatalf("expected user under excluded base DN to have no department, got %q", got)
	}
}

func TestLDAPDepartmentsCanExcludeBaseDNDepartment(t *testing.T) {
	includeBase := false
	s := &LDAPSyncer{config: &LDAPConfig{
		BaseDN:        "ou=公司,dc=example,dc=com",
		IncludeBaseDN: &includeBase,
	}}
	entries := []*ldap.Entry{
		ldap.NewEntry("ou=公司,dc=example,dc=com", map[string][]string{
			"ou":        {"公司"},
			"entryUUID": {"company-entry-uuid"},
		}),
		ldap.NewEntry("ou=研发部,ou=公司,dc=example,dc=com", map[string][]string{
			"ou":        {"研发部"},
			"entryUUID": {"dept-entry-uuid"},
		}),
	}

	got := s.departmentsFromEntries(entries)

	if len(got) != 1 {
		t.Fatalf("expected only child department, got %#v", got)
	}
	if got[0].Name != "研发部" {
		t.Fatalf("expected child department, got %q", got[0].Name)
	}
	if got[0].ParentID != "" {
		t.Fatalf("expected excluded base DN parent to be empty, got %q", got[0].ParentID)
	}
	if got[0].Level != 0 {
		t.Fatalf("expected child department level to start at 0 after excluding base DN, got %d", got[0].Level)
	}
}
