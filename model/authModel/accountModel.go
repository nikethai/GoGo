package authModel

type Account struct {
	ID       string `json:"id" bson:"_id"`
	Uuid     string `json:"uuid" bson:"uuid"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
	Role     []Role `json:"roles" bson:"roles,inline"`
}
