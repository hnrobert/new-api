package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelGetUserIds(t *testing.T) {
	tests := []struct {
		name    string
		userIds string
		want    []int
	}{
		{"empty", "", []int{}},
		{"whitespace only", "   ", []int{}},
		{"single", "42", []int{42}},
		{"comma list", "1,2,3", []int{1, 2, 3}},
		{"with spaces", " 1 , 2 , 3 ", []int{1, 2, 3}},
		{"dedup", "1,2,1,3,2", []int{1, 2, 3}},
		{"skips non-numeric", "1,abc,3", []int{1, 3}},
		{"trailing comma", "1,2,", []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &Channel{UserIds: tt.userIds}
			got := ch.GetUserIds()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChannelIsUserAllowed(t *testing.T) {
	// 核心访问控制契约：未设限制对所有人开放；设了限制仅授权用户可用，匿名一律拒绝。
	tests := []struct {
		name    string
		userIds string
		userId  int
		want    bool
	}{
		{"unrestricted user", "", 5, true},
		{"unrestricted anonymous", "", 0, true},
		{"restricted owner", "5,9", 5, true},
		{"restricted other user", "5,9", 7, false},
		{"restricted anonymous rejected", "5,9", 0, false},
		{"restricted negative rejected", "5,9", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &Channel{UserIds: tt.userIds}
			assert.Equal(t, tt.want, ch.IsUserAllowed(tt.userId))
		})
	}
}

func TestChannelIsUserRestricted(t *testing.T) {
	assert.False(t, (&Channel{UserIds: ""}).IsUserRestricted())
	assert.False(t, (&Channel{UserIds: "   "}).IsUserRestricted())
	assert.True(t, (&Channel{UserIds: "5"}).IsUserRestricted())
	assert.True(t, (&Channel{UserIds: "5,9"}).IsUserRestricted())
}

// TestGetGroupEnabledModelsForUserUserFilter 验证 user 过滤分支与 userId<=0 直通分支
// 在内存 DB（SQLite）下都正确，且不依赖任何 SQL 字符串包含函数。
func TestGetGroupEnabledModelsForUserUserFilter(t *testing.T) {
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)

	chOwn := &Channel{
		Id:      1,
		Type:    1,
		Status:  1,
		Models:  "gpt-4o-mini",
		Group:   "default",
		UserIds: "100",
	}
	chPublic := &Channel{
		Id:      2,
		Type:    1,
		Status:  1,
		Models:  "gpt-4o",
		Group:   "default",
		UserIds: "",
	}
	require.NoError(t, DB.Create(chOwn).Error)
	require.NoError(t, DB.Create(chPublic).Error)
	require.NoError(t, chOwn.AddAbilities(nil))
	require.NoError(t, chPublic.AddAbilities(nil))

	// userId<=0（内部/管理端）：不做 user 过滤，返回两个模型。
	all := GetGroupEnabledModelsForUser([]string{"default"}, 0)
	assert.ElementsMatch(t, []string{"gpt-4o-mini", "gpt-4o"}, all)

	// owner 100：两个模型都可见（私有渠道对其开放）。
	owner := GetGroupEnabledModelsForUser([]string{"default"}, 100)
	assert.ElementsMatch(t, []string{"gpt-4o-mini", "gpt-4o"}, owner)

	// 非 owner 200：私有模型被隐藏，仅剩公共模型。
	other := GetGroupEnabledModelsForUser([]string{"default"}, 200)
	assert.ElementsMatch(t, []string{"gpt-4o"}, other)
}
