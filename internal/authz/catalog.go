package authz

type Perm string

const (
	AdminView         Perm = "admin.view"
	SeriesManage      Perm = "series.manage"
	SeriesView        Perm = "series.view"
	SeriesEdit        Perm = "series.edit"
	ChapterPublish    Perm = "chapter.publish"
	ChapterManage     Perm = "chapter.manage"
	ChapterView       Perm = "chapter.view"
	ChapterSetAccess  Perm = "chapter.set_access"
	ContentReadVIP    Perm = "content.read_vip"
	MediaManage       Perm = "media.manage"
	MediaUpload       Perm = "media.upload"
	UserView          Perm = "user.view"
	UserBan           Perm = "user.ban"
	UserManage        Perm = "user.manage"
	UserAssignRole    Perm = "user.assign_role"
	RoleManage        Perm = "role.manage"
	RoleAssign        Perm = "role.assign"
	RoleView          Perm = "role.view"
	VIPManage         Perm = "vip.manage"
	VIPView           Perm = "vip.view"
	ReportHandle      Perm = "report.handle"
	ReportView        Perm = "report.view"
	AuditView         Perm = "audit.view"
	InstanceConfigure Perm = "instance.configure"
	SystemView        Perm = "system.view"
	SearchManage      Perm = "search.manage"
	SearchView        Perm = "search.view"
	SettingsManage    Perm = "settings.manage"
	SettingsView      Perm = "settings.view"
)

type ScopeKind int

const (
	Global ScopeKind = iota
	Group
)

type Def struct {
	Key         Perm
	Description string
	Scopes      []ScopeKind
	RequiresMFA bool
}

var Catalog = []Def{
	{AdminView, "Acesso ao painel administrativo", []ScopeKind{Global}, true},
	{SeriesManage, "Criar, editar e excluir séries", []ScopeKind{Global, Group}, false},
	{SeriesView, "Visualizar séries", []ScopeKind{Global, Group}, false},
	{SeriesEdit, "Editar séries", []ScopeKind{Global, Group}, false},
	{ChapterPublish, "Publicar capítulos", []ScopeKind{Global, Group}, false},
	{ChapterManage, "Gerenciar capítulos", []ScopeKind{Global, Group}, false},
	{ChapterView, "Visualizar capítulos", []ScopeKind{Global, Group}, false},
	{ChapterSetAccess, "Definir acesso de capítulos", []ScopeKind{Global, Group}, true},
	{ContentReadVIP, "Ler conteúdo VIP", []ScopeKind{Global, Group}, false},
	{MediaManage, "Gerenciar mídia", []ScopeKind{Global, Group}, false},
	{MediaUpload, "Fazer upload de mídia", []ScopeKind{Global, Group}, false},
	{UserView, "Visualizar usuários", []ScopeKind{Global}, false},
	{UserBan, "Banir usuários", []ScopeKind{Global}, true},
	{UserManage, "Gerenciar usuários", []ScopeKind{Global}, true},
	{UserAssignRole, "Atribuir roles a usuários", []ScopeKind{Global, Group}, true},
	{RoleManage, "Criar/editar roles", []ScopeKind{Global}, true},
	{RoleAssign, "Atribuir roles", []ScopeKind{Global, Group}, true},
	{RoleView, "Visualizar roles", []ScopeKind{Global}, false},
	{VIPManage, "Gerenciar assinaturas VIP", []ScopeKind{Global}, true},
	{VIPView, "Visualizar assinaturas VIP", []ScopeKind{Global}, false},
	{ReportHandle, "Processar denúncias", []ScopeKind{Global, Group}, false},
	{ReportView, "Visualizar denúncias", []ScopeKind{Global, Group}, false},
	{AuditView, "Visualizar logs de auditoria", []ScopeKind{Global}, true},
	{InstanceConfigure, "Configurar a instância", []ScopeKind{Global}, true},
	{SystemView, "Visualizar status do sistema", []ScopeKind{Global}, false},
	{SearchManage, "Gerenciar configurações de busca", []ScopeKind{Global}, false},
	{SearchView, "Visualizar configurações de busca", []ScopeKind{Global}, false},
	{SettingsManage, "Gerenciar configurações", []ScopeKind{Global}, true},
	{SettingsView, "Visualizar configurações", []ScopeKind{Global}, false},
}

func (d Def) allowsScope(k ScopeKind) bool {
	for _, s := range d.Scopes {
		if s == k {
			return true
		}
	}
	return false
}

func CatalogMap() map[Perm]Def {
	m := make(map[Perm]Def, len(Catalog))
	for _, d := range Catalog {
		m[d.Key] = d
	}
	return m
}