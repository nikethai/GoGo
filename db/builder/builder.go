package builder

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type IBuilder interface {
	Search() interface{}
	Sort() interface{}
	Pagination() interface{}
	Lookup() interface{}
	Unwind() interface{}
	Group() interface{}
	Project() interface{}
}

func Search(field string, searchVal string, options string) bson.M {
	return bson.M{"$match": bson.M{field: bson.M{"$regex": searchVal, "$options": options}}}
}

func SearchInsensitiveMultiline(field string, searchVal string) bson.M {
	return bson.M{"$match": bson.M{field: bson.M{"$regex": searchVal, "$options": "im"}}}
}

// mongo id
func SearchById(field string, id primitive.ObjectID) bson.M {
	return bson.M{"$match": bson.M{field: id}}
}

// ascending 1, descending -1
func Sort(field string, order string) bson.M {
	return bson.M{"$sort": bson.M{field: order}}
}

// Skip: number of documents to skip
// Limit: number of documents to return
func Pagination(page int, limit int) (bson.M, bson.M) {
	return bson.M{"$skip": (page - 1) * limit}, bson.M{"$limit": limit}
}

/*
* Lookup
* @param from: collection name in db
* @param localField: field name of children document
* @param foreignField: field name of parent document
* @param as: new field name in result
 */
func Lookup(from string, localField string, foreignField string, as string) bson.M {
	return bson.M{"$lookup": bson.M{
		"from":         from,
		"localField":   localField,
		"foreignField": foreignField,
		"as":           as,
	}}
}

func Unwind(field string) bson.M {
	return bson.M{"$unwind": "$" + field}
}

func Group(id string, fields []string, operators []string) bson.M {
	group := bson.M{"_id": "$" + id}
	for i, field := range fields {
		group[field] = bson.M{operators[i]: "$" + field}
	}
	return bson.M{"$group": group}
}

func Project(fields []string) bson.M {
	project := bson.M{}
	for _, field := range fields {
		project[field] = 1
	}
	return bson.M{"$project": project}
}

func ConvertToObjectId(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

func GetAll[T any](collection *mongo.Collection) (*[]T, error) {
	var result []T
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func GetById[T any](collection *mongo.Collection, pid string) (*T, error) {
	var result T
	id, err := ConvertToObjectId(pid)
	if err != nil {
		return nil, err
	}
	err = collection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: id}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func GetByField[T any](collection *mongo.Collection, field string, value interface{}) (*T, error) {
	var result T
	err := collection.FindOne(context.TODO(), bson.D{{Key: field, Value: value}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
