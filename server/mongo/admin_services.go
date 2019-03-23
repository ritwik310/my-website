package mongo

import (
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// "github.com/ritwik310/my-website/server/auth"

// AdminService ...
type AdminService struct {
	collection *mgo.Collection
}

// NewAdminService ...
func NewAdminService(session *Session, dbName string, collectionName string) *AdminService {
	collection := session.GetCollection(dbName, collectionName)
	fmt.Println("collection ==", collection)
	return &AdminService{collection: collection}
}

// Create ...
func (as *AdminService) Create(a *Admin) error {
	admin := newAdminModel(a)
	fmt.Printf("aaa => %v", a)
	return as.collection.Insert(&admin)
}

// Get ...
func (as *AdminService) Get(Email string, ID string) (*Admin, error) {
	model := adminModel{}
	err := as.collection.Find(bson.M{"email": Email, "googleid": ID}).One(&model)
	return model.toAdmin(), err
}
