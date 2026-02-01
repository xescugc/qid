package user

type User struct {
	ID       uint32 `json:"id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
	Password string `json:"-"`
	Admin    bool   `json:"admin"`
}

type WithMemberships struct {
	User

	Memberships []Member
}

type Member struct {
	Admin         bool
	TeamCanonical string
}

func (u *WithMemberships) IsAdmin(tcs ...string) bool {
	if u.Admin {
		return true
	}
	for _, tc := range tcs {
		for _, m := range u.Memberships {
			if m.Admin && m.TeamCanonical == tc {
				return true
			}
		}
	}
	return false
}
