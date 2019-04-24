package cms

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func GradesAsCSV(c *gin.Context) {
	aid, _ := c.Get("aid")
	cid, _ := c.Get("cid")

	file, filename, numBytes, err := cm.GetGradesAsCSV(aid, cid)
	if err != nil {
		fmt.Println("err", err)
	}

	additonalHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="%s.csv"`, filename),
	}

	c.DataFromReader(200, numBytes, "text/csv", file, additonalHeaders)
}
