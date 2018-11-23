package auth

import (
	ctx "context"
	"errors"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/goware/emailx"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/options"
	"golang.org/x/crypto/bcrypt"

	"backend/models"

	"github.com/stevens-tyr/tyr-gin"
)

// isValidEmail checks an email string to be valid and with resolvable host.
func isValidEmail(email string) error {
	err := emailx.Validate(email)
	if err != nil {
		if err == emailx.ErrInvalidFormat {
			return models.ErrorEmailNotValid
		}
		if err == emailx.ErrUnresolvableHost {
			return models.ErrorUnresolvableEmailHost
		}
		return err
	}
	return nil
}

// Register a function that registers a User.
func Register(c *gin.Context) {
	var register models.RegisterForm
	err := c.ShouldBindJSON(&register)
	if !tyrgin.ErrorHandler(err, c, 400, gin.H{
		"status_code": 400,
		"message":     "Incorrect json format.",
		"error":       err,
	}) {
		return
	}

	db, err := tyrgin.GetMongoDB(os.Getenv("DB_NAME"))
	if !tyrgin.ErrorHandler(err, c, 500, gin.H{
		"status_code": 500,
		"message":     "Failed to get Mongo Session.",
		"error":       err,
	}) {
		return
	}

	col := tyrgin.GetMongoCollection("users", db)

	if err = isValidEmail(register.Email); err != nil {
		msg := "Email is invalid"
		if err == models.ErrorUnresolvableEmailHost {
			msg = "Unable to resolve email host"
		}
		tyrgin.ErrorHandler(err, c, 400, gin.H{
			"status_code": 400,
			"message":     msg,
			"error":       err,
		})
		return
	}

	var user models.User
	res := col.FindOne(ctx.Background(), bson.M{"email": register.Email}, options.FindOne())
	res.Decode(&user)

	if user.Email != "" {
		err = errors.New("Email is taken")
		tyrgin.ErrorHandler(err, c, 400, gin.H{
			"status_code": 400,
			"message":     "Email is taken.",
			"error":       err,
		})
		return
	}

	if register.Password != register.PasswordConfirmation {
		tyrgin.ErrorHandler(errors.New("Non Matching Passwords"), c, 400, gin.H{
			"status_code": 400,
			"message":     "Your password and password confirmation do not match.",
			"error":       err,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(register.Password), bcrypt.DefaultCost)
	if !tyrgin.ErrorHandler(err, c, 500, gin.H{
		"status_code": 500,
		"message":     "Failed to generate hash",
		"error":       err,
	}) {
		return
	}

	user = models.User{
		Email:           register.Email,
		Password:        hash,
		First:           register.First,
		Last:            register.Last,
		EnrolledCourses: make([]models.EnrolledCourse, 0),
	}

	_, err = col.InsertOne(ctx.Background(), &user, options.InsertOne())
	if !tyrgin.ErrorHandler(err, c, 500, gin.H{
		"status_code": 500,
		"message":     "Failed to create user.",
		"error":       err,
	}) {
		return
	}

	fmt.Println("end")
	c.JSON(200, gin.H{
		"status_code": 200,
		"message":     "User created.",
	})

}
