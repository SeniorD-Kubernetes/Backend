package assignmentmodels

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"

	"backend/errors"
	"backend/forms"

	tyrgin "github.com/stevens-tyr/tyr-gin"
)

type (
	AssignmentSubmission struct {
		UserID        primitive.ObjectID `bson:"userID" json:"userID" binding:"required"`
		SubmissionID  primitive.ObjectID `bson:"submissionID" json:"submissionID" binding:"required"`
		AttemptNumber int                `bson:"attemptNumber" json:"attemptNumber" binding:"required"`
	}

	Test struct {
		Name           string `bson:"name" json:"name" binding:"required"`
		ExpectedOutput string `bson:"expectedOutput" json:"expectedOutput" binding:"required"`
		StudentFacing  bool   `bson:"studentFacing" json:"studentFacing" binding:"exists"`
		TestCMD        string `bson:"testCMD" json:"testCMD" binding:"required"`
	}

	// MongoAssignment struct to store information about an assignment.
	MongoAssignment struct {
		ID              primitive.ObjectID     `bson:"_id" form:"id" json:"id"`
		Language        string                 `bson:"language" form:"language" binding:"required" json:"language"`
		Version         string                 `bson:"version" form:"version" binding:"required" json:"version"`
		Name            string                 `bson:"name" form:"name" binding:"required" json:"name"`
		NumAttempts     int                    `bson:"numAttempts" form:"numAttempts" binding:"required" json:"numAttempts"`
		Description     string                 `bson:"description" form:"description" binding:"required" json:"description"`
		DueDate         primitive.DateTime     `bson:"dueDate" form:"dueDate" binding:"required" json:"dueDate"`
		Published       bool                   `bson:"published" form:"published" binding:"required" json:"-"`
		SupportingFiles primitive.ObjectID     `bson:"supportingFiles" form:"supportingFiles" json:"supportingFiles"`
		TestBuildCMD    string                 `bson:"testBuildCMD" form:"testBuildCMD" json:"testBuildCMD"`
		Tests           []Test                 `bson:"tests" form:"tests" binding:"required" json:"tests"`
		Submissions     []AssignmentSubmission `bson:"submissions" form:"submissions" json:"submissions"`
	}

	AssignmentInterface struct {
		ctx context.Context
		col *mongo.Collection
	}
)

func New() *AssignmentInterface {
	db, _ := tyrgin.GetMongoDB(os.Getenv("DB_NAME"))
	col := tyrgin.GetMongoCollection("assignments", db)

	return &AssignmentInterface{
		context.Background(),
		col,
	}
}

func (a *AssignmentInterface) Create(form forms.CreateAssignmentPostForm, cid string) (*primitive.ObjectID, *primitive.ObjectID, errors.APIError) {
	tests := make([]Test, len(form.Tests))
	for index := range form.Tests {
		tests[index] = Test(form.Tests[index])
	}

	aid := primitive.NewObjectID()
	supportingFiles := primitive.NewObjectID()
	assign := MongoAssignment{
		ID:              aid,
		Language:        form.Language,
		Version:         form.Version,
		Name:            form.Name,
		NumAttempts:     form.NumAttempts,
		Description:     form.Description,
		SupportingFiles: supportingFiles,
		DueDate:         form.DueDate,
		Published:       false,
		TestBuildCMD:    form.TestBuildCMD,
		Tests:           tests,
		Submissions:     make([]AssignmentSubmission, 0),
	}

	_, err := a.col.InsertOne(a.ctx, assign, options.InsertOne())
	if err != nil {
		return nil, nil, errors.ErrorDatabaseFailedCreate
	}

	return &aid, &supportingFiles, nil
}

func (a *AssignmentInterface) Delete(aid interface{}) errors.APIError {
	_, err := a.col.DeleteOne(a.ctx, bson.M{"_id": aid}, options.Delete())
	if err != nil {
		return errors.ErrorDatabaseFailedDelete
	}

	return nil
}

func (a *AssignmentInterface) Get(aid interface{}) (*MongoAssignment, errors.APIError) {
	var assign *MongoAssignment
	res := a.col.FindOne(a.ctx, bson.M{"_id": aid}, options.FindOne())

	err := res.Decode(&assign)
	if err != nil {
		return nil, errors.ErrorInvalidBSON
	}

	return assign, nil
}

func (a *AssignmentInterface) Update(assign MongoAssignment) errors.APIError {
	_, err := a.col.UpdateOne(
		a.ctx,
		bson.M{
			"_id": assign.ID,
		},
		bson.M{
			"$set": bson.M{
				"language":     assign.Language,
				"version":      assign.Version,
				"name":         assign.Name,
				"description":  assign.Description,
				"dueDate":      assign.DueDate,
				"published":    assign.Published,
				"testBuildCMD": assign.TestBuildCMD,
				"tests":        assign.Tests,
				"numAttempts":  assign.NumAttempts,
			},
		},
	)
	if err != nil {
		return errors.ErrorDatabaseFailedUpdate
	}

	return nil
}

func (a *AssignmentInterface) GetAsFile(aid interface{}) (*MongoAssignment, errors.APIError) {
	var assign *MongoAssignment
	res := a.col.FindOne(a.ctx, bson.M{"_id": aid}, options.FindOne())

	err := res.Decode(&assign)
	if err != nil {
		return nil, errors.ErrorInvalidBSON
	}

	return assign, nil
}

func (a *AssignmentInterface) GetFull(aid, uid interface{}, role string) (map[string]interface{}, errors.APIError) {
	query := []interface{}{
		bson.M{"$match": bson.M{"_id": aid}},
	}

	if role == "student" {
		query = append(query, bson.M{
			"$lookup": bson.M{
				"from":         "submissions",
				"localField":   "submissions.submissionID",
				"foreignField": "_id",
				"as":           "submissions",
			},
		})
	} else {
		query = append(query, bson.M{
			"$lookup": bson.M{
				"from": "submissions",
				"let":  bson.M{"ass": "$_id"},
				"as":   "studentSubmissions",
				"pipeline": bson.A{
					bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$$ass", "$assignmentID"}}}},
					bson.M{"$sort": bson.M{"submissionDate": 1}},
					bson.M{"$group": bson.M{"_id": "$userID", "submissions": bson.M{"$push": "$$ROOT"}}},
					bson.M{
						"$lookup": bson.M{
							"from":         "users",
							"localField":   "_id",
							"foreignField": "_id",
							"as":           "student",
						},
					},
					bson.M{
						"$project": bson.M{
							"_id":         1,
							"submissions": 1,
							"student":     bson.M{"$arrayElemAt": bson.A{"$student", 0}},
						},
					},
					bson.M{
						"$project": bson.M{
							"_id":               0,
							"submissions":       1,
							"student.email":     1,
							"student.firstName": 1,
							"student.lastName":  1,
						},
					},
				},
			},
		})
	}

	project := bson.M{
		"$project": bson.M{
			"_id":             1,
			"language":        1,
			"version":         1,
			"name":            1,
			"numAttempts":     1,
			"description":     1,
			"supportingFiles": 1,
			"dueDate":         1,
			"published":       1,
			"testBuildCMD":    1,
			"tests":           1,
		},
	}

	if role == "student" {
		project["$project"].(primitive.M)["submissions"] = bson.M{
			"$filter": bson.M{
				"input": "$submissions",
				"as":    "submission",
				"cond":  bson.M{"$eq": bson.A{"$$submission.userID", uid.(primitive.ObjectID)}},
			},
		}
		project["$project"].(primitive.M)["tests"] = bson.M{
			"$filter": bson.M{
				"input": "$tests",
				"as":    "test",
				"cond":  "$$test.studentFacing",
			},
		}
		query = append(query, project, bson.M{
			"$match": bson.M{
				"$expr": bson.M{"$eq": bson.A{"$published", true}}},
		})
	} else {
		project["$project"].(primitive.M)["studentSubmissions"] = 1
		query = append(query, project)
	}

	var assign map[string]interface{}
	cur, err := a.col.Aggregate(a.ctx, query, options.Aggregate())
	if err != nil {
		return nil, errors.ErrorInvalidBSON
	}

	for cur.Next(a.ctx) {
		err = cur.Decode(&assign)
		if err != nil {
			return nil, errors.ErrorResourceNotFound
		}
	}

	return assign, nil
}

func (a *AssignmentInterface) LatestUserSubmission(aid, uid interface{}) (*MongoAssignment, int, errors.APIError) {
	assignment, err := a.Get(aid)
	if err != nil {
		return nil, 0, err
	}

	attempt := 0
	for _, assignSub := range assignment.Submissions {
		if assignSub.UserID == uid.(primitive.ObjectID) && assignSub.AttemptNumber > attempt {
			attempt = assignSub.AttemptNumber
		}
	}

	return assignment, attempt, nil
}

func (a *AssignmentInterface) InsertSubmission(aid, uid, sid interface{}, attempt int) errors.APIError {
	insert := AssignmentSubmission{
		UserID:        uid.(primitive.ObjectID),
		SubmissionID:  sid.(primitive.ObjectID),
		AttemptNumber: attempt,
	}

	_, err := a.col.UpdateOne(
		a.ctx,
		bson.M{"_id": aid},
		bson.M{"$push": bson.M{"submissions": &insert}},
		options.Update(),
	)
	if err != nil {
		return errors.ErrorDatabaseFailedUpdate
	}

	return nil
}

func (a *AssignmentInterface) DeleteSubmission(aid, sid interface{}) errors.APIError {
	_, err := a.col.UpdateOne(
		a.ctx,
		bson.M{"_id": aid},
		bson.M{"$pull": bson.M{"submissions": bson.M{"submissionID": sid.(primitive.ObjectID)}}},
		options.Update(),
	)
	if err != nil {
		return errors.ErrorDatabaseFailedUpdate
	}

	return nil
}

func (a *AssignmentInterface) AsFile(aid interface{}) (*bytes.Reader, string, int64, errors.APIError) {
	var jsonBytes []byte
	assignment, err := a.GetAsFile(aid)
	if err != nil {
		return nil, "", 0, err
	}

	jsonBytes, errs := json.Marshal(assignment)
	if errs != nil {
		return nil, "", 0, errors.ErrorFailedToConvertStructToJSON
	}

	return bytes.NewReader(jsonBytes), assignment.Name, int64(len(jsonBytes)), nil
}
