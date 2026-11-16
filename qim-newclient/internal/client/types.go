// 会话 / 消息模型与 app 状态结构。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"context"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/model"
	"qim-newclient/internal/qapi"
	"sync"
)

// displayName 侧栏头部显示的用户名：登录后由服务端返回的昵称填充。
func (a *app) displayName() string {
	if a.userName != "" {
		return a.userName
	}
	return "未登录"
}

// 数据模型别名：真正的定义在 internal/model，这里保留原名让业务代码无需改动。
type (
	conversation = model.Conversation
	member       = model.Member
	message      = model.Message
	messageQuote = model.MessageQuote
)

// app 是客户端状态，view 由它生成。
type app struct {
	// mu 只在无窗口的 headless 测试里承担同步：那时没有 UI 线程可以投递，
	// mutate 直接执行，网络回调与测试读取必须靠它串行化。
	// 真窗口下所有状态变更都经 Window.Update 回到主线程，天然串行。
	mu          sync.RWMutex
	win         *mygo.Window
	convs       []conversation
	active      int // 当前选中的会话下标，-1 = 未选择
	msgs        []message
	input       string
	search      string
	list        ui.ListState
	convList    ui.ListState // 侧栏会话列表的滚动状态（按会话 id 保位）
	groupsList  ui.ListState // 侧栏群聊列表的滚动状态
	memberList  ui.ListState // 信息栏群成员列表的滚动状态
	forwardList ui.ListState // 转发目标选择列表的滚动状态
	nextID      int

	// 转发 / 合并转发（对齐 qim-client 右键「转发」与多选「合并转发」）：
	// 打开面板选目标会话，把 forwardReq 发到目标会话，不改当前活跃会话。
	// forwardReq 为 nil 时按文本转发（兼容搜索结果等无原始消息的入口）。
	forwardOpen  bool
	forwardText  string
	forwardTitle string
	forwardReq   *qapi.SendRequest

	// 群公告编辑（对齐 qim-client「编辑群公告」）：仅群主/管理员可见入口，
	// annDraft 是编辑框草稿，保存走 PUT /groups/:convID/announcement，失败回滚。
	annEditorOpen bool
	annDraft      string

	// 头部「更多」菜单的两个动作（原版在 ChatHeaderActions 的下拉菜单里）：
	// 修改群名称走 PUT /groups/:id，解散群聊走 DELETE /groups/:id（先二次确认）。
	nameEditorOpen      bool
	nameDraft           string
	dissolveConfirmOpen bool

	// 会话列表右键的破坏性动作要先二次确认：「退出群聊」走
	// POST /groups/:id/exit，「移除会话」走 DELETE /conversations/:id。
	// kind 为空表示没有弹层；名称只用于弹层文案，id 是发起时的快照。
	convConfirmKind string
	convConfirmID   string
	convConfirmName string

	// 已读详情弹层（对齐 qim-client ReadUsersModal）：右键自己的消息「查看已读」，
	// 拉取 read_users 列表展示；未读数 = total_members - read_count。
	readUsersOpen bool
	readUsersBusy bool
	readUsersID   string
	readUsersInfo *qapi.MessageReadInfo

	// 网络：Go 是唯一数据源，token 由 qapi.Client 持有，UI 只读状态。
	client     *qapi.Client
	rt         *qapi.Realtime
	ctx        context.Context
	cancel     context.CancelFunc
	loggedIn   bool
	userName   string
	selfID     string
	connStatus qapi.Status

	// 原生登录表单
	loginServer  string
	loginUser    string
	loginPass    string
	loginCode    string
	loginErr     string
	loginBusy    bool
	need2FA      bool
	twoFASession string

	// 界面上的临时状态
	notice       string            // 最近一次错误/提示
	loadingMsgs  bool              // 正在拉第一页历史消息
	loadingOlder bool              // 正在翻更早的一页
	hasOlder     bool              // 服务端还有更早的消息
	oldestID     string            // 当前列表里最早一条服务端消息 id（before_id 游标）
	drafts       map[string]string // 每个会话没发出去的输入

	// 消息操作（P1 第二批）：引用回复、右键菜单操作、多选、未读分隔线。
	// 这些状态都只在当前会话内有效，切会话/退出时整组清掉。
	quote         *messageQuote
	selectMode    bool
	selected      map[string]bool
	pendingUnread int // 进入会话那一刻的未读数，历史加载完用来插分隔线

	// @ 提及：输入框里保持 "@名字" 可读文本，发送前才替换成后端 token。
	// mentionPicks 按会话随草稿一起存，切回来才不会把 @ 变成普通文本。
	mentionPicks  map[string]string
	mentionDrafts map[string]map[string]string
	mentionQuery  string
	lastMentionQ  string
	mentionHits   []qapi.SearchUserHit
	mentionBusy   bool
	mentionSeq    uint64

	// 搜索：三路结果来自服务端，本地会话过滤照旧。
	// searchSeq 用原子序号丢弃过期响应，避免打字快时旧结果盖掉新结果。
	searchHits qapi.SearchHits
	searchBusy bool
	searchErr  string
	searchSeq  uint64
	lastSearch string
	highlight  string // 搜索结果跳转后要描边的那条消息 key

	// @ 我横幅（对齐 qim-client AtMentionBanner）：消息列表里 atMe 且
	// 未被横幅消费过的消息计数；点击定位第一条并记入 atSeen。
	atSeen map[string]bool

	// 待发送文件（对齐 qim-client pendingFiles）：粘贴截图 / 拖拽文件先进
	// 这个队列，composer 上方出预览条，Enter 发送，✕ 移除。
	pendingFiles []string
	svc          ui.Services // OnPaste 回调里读剪贴板图片用（build 时捕获）

	// 登录态持久化：store 可替换（headless 测试用内存实现），
	// saved 是最近一次写入的快照，refresh 轮换 token 时只改其中两个字段。
	store   sessionStore
	savedMu sync.Mutex
	saved   savedSession

	// shot 是微信式截图的覆盖层，标题栏按钮和全局快捷键都调它。
	shot *overlay

	// P2 富媒体：图片位图缓存、附件下载状态。自带锁，不与 a.mu 嵌套。
	media mediaCache

	// P4 系统集成：新消息桌面通知。notifier 可注入 fake 便于测试；
	// notifyOnNew 是用户开关（设置页或默认开启），关掉后不再弹通知。
	notifier    notifier
	notifyOnNew bool

	// P4 设置页：打开时 view 整屏切到设置视图，替代三栏。
	settingsOpen bool

	// 通知中心（对齐 qim-client NotificationCenter.vue）：侧栏铃铛打开面板，
	// 列表分页加载（pageSize=50），支持已读/删除/置顶/重要/一键已读/清空与
	// action_type 动作按钮；WS new_notification 实时插队并累计未读徽标。
	notifsOpen    bool
	notifsBusy    bool
	notifs        []qapi.Notification
	notifsTab     string // all / unread / important
	notifsPage    int    // 已加载到的页码，0 = 尚未拉过
	notifsHasMore bool
	unreadNotifs  int
	notifList     ui.ListState

	// 频道（对齐 qim-client stores/channel.ts）：整屏视图 = 左栏频道列表
	// （订阅/广场 tab + 搜索 + 创建）+ 右侧消息卡片流（点赞/评论）。
	// 未读由 WS "notification"（channel_message）驱动，本地维护。
	channelsOpen      bool
	channelsLoaded    bool
	channels          []qapi.Channel
	channelTab        string // subscribed / discover
	channelSearch     string
	channelSel        int
	chMsgs            []qapi.ChannelMessage
	chMsgsLoading     bool
	chInput           string
	chBusy            bool
	chCreateOpen      bool
	chName            string
	chDesc            string
	chCommentFor      string // 展开评论的消息 id（"" = 收起）
	chComments        []qapi.ChannelComment
	chCommentsLoading bool
	chCommentInput    string
	chCommentBusy     bool

	// B 组：个人资料 / 用户卡 / 组织架构建群 / 在线状态
	// （对齐 qim-client SelfProfileModal / UserProfileCard / CreateGroupModal+OrgTreePicker）。
	me            *qapi.User // 当前登录用户完整资料（打开资料弹层时刷新）
	meDept        string     // 自己所在部门名（组织树反查）
	profileOpen   bool       // 自我资料弹层
	profileBusy   bool       // 拉资料中
	profileSaving bool       // 保存中
	profNick      string     // 昵称草稿
	profSign      string     // 签名草稿
	avatarBusy    bool       // 头像上传中

	userCardOpen bool       // 用户资料小卡
	userCardBusy bool       // 详情加载中
	userCardUser *qapi.User // 兜底：先给名字，详情回来再覆盖
	userCardDept string     // 部门名（组织树反查）
	userCardPos  string     // 职位（树接口不含，预留）

	orgOpen     bool              // 建群弹窗
	orgLoading  bool              // 组织树加载中
	orgDepts    []qapi.Department // 组织树缓存
	orgLoaded   bool
	orgExpanded map[string]bool      // 部门展开状态
	orgName     string               // 群名
	orgSearch   string               // 成员搜索
	orgSel      map[string]bool      // 选中成员 id → true
	orgUserByID map[string]qapi.User // 已选成员详情（渲染胶囊栏）
	orgBusy     bool                 // 创建中

	// 侧边栏页签（对齐 SideOptions 的 activeOption）：recent 会话 /
	// org 组织架构树 / groups 群聊列表。应用页签按需求搁置。
	tab            string
	orgTabExpanded map[string]bool // 组织架构页签的部门展开（与建群选择器的 orgExpanded 分开）

	// 群成员侧栏（对齐 MemberSidebar.vue）：仅群聊显示，180px 展开 /
	// 48px 折叠头像栏，默认展开（collapsed=false）。
	membersCollapsed bool
	memberSearchOpen bool
	memberSearch     string

	// 邀请成员：复用建群弹窗的组织树选人（orgMode = create / invite）。
	orgMode    string
	inviteConv string

	statusMap map[string]string // user_id → online/offline（WS + 批量接口维护）

	// A 组聊天体验（对齐 qim-client EmojiPanel / SlashCommandPanel / quickCommand）：
	// emoji 面板（工具栏笑脸开合，两 tab）；/quick 快捷回复（输入 /quick 触发，
	// 短语来自 /user-settings/quick_replies，未自定义兜底默认短语）。
	emojiOpen  bool
	emojiTab   string // "" = 默认表情 / "classic" = 经典表情
	emojiList  ui.ListState
	quickOpen  bool
	quickBusy  bool
	quickItems []string
	quickList  ui.ListState

	// 斜杠命令首次引导条（对齐 SlashCommandHint.vue）：hint 存「已关闭」
	// 偏好，关闭后重启不再出现。nil 时按内存实现处理，测试不用造存储。
	hint hintStore

	// C 组杂项（对齐 qim-client FeedbackModal / Main.vue 检查更新）：
	// 问题反馈弹层（提交/我的反馈两视图）与检查更新弹窗。
	feedbackOpen bool
	fbView       string // submit / list
	fbType       string // bug / feature / other
	fbContent    string
	fbShotPath   string
	fbBusy       bool
	fbErr        string
	fbList       []qapi.UserFeedback
	fbLoading    bool
	fbListPage   int
	fbHasMore    bool
	fbExpanded   int64
	fbListState  ui.ListState

	updOpen     bool
	updChecking bool
	updHasNew   bool
	updResult   string
	updInfo     *qapi.ClientVersion

	// 消息渲染对齐：合并转发记录弹窗 / share 原文弹窗
	mfOpen    bool
	mfView    *qapi.MergedForward
	mfList    ui.ListState
	shareOpen bool
	shareView *qapi.ShareCard

	// D 组文件网盘（对齐 FileManagementApp.vue）：文件箱整屏视图。
	filesOpen       bool
	filesLoaded     bool
	filesLoading    bool
	files           []qapi.FileItem
	filesTotal      int
	filesFolderID   int64 // -1 = 根目录
	filesCrumb      []qapi.FolderItem
	treeRoot        []qapi.FolderItem
	treeKids        map[int64][]qapi.FolderItem
	treeExpanded    map[int64]bool
	filesStarred    bool
	filesSortIdx    int
	filesSearch     string
	filesSearchQ    string
	filesSel        map[int64]bool
	filesBusy       bool
	filesModal      string // "" / "create" / "move" / "rename"
	newFolderName   string
	renameID        int64
	renameName      string
	moveSel         int64 // 0 = 根目录
	moveOneID       int64 // >0 = 移动单个文件
	uploadTasks     []*fileUploadTask
	uploadSeq       int
	filesUsageUsed  int64
	filesUsageQuota int64
	filesListState  ui.ListState
}
