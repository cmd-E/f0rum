package tools

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/IrvinIrvin/forum/app/contentmanager/postsmanager"
)

// Templates stores all templates
var Templates *template.Template

// GetID thread id from url
func GetID(path string) string {
	temp := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return temp
		}
		temp = string(path[i]) + temp
	}
	return temp
}

type dataerr struct {
	Errcode    string
	Errmessage string
}

// ExecuteError executes error template with given error code
func ExecuteError(w http.ResponseWriter, Errcode int, Errormessage string) {
	var de dataerr
	if Errcode != 0 { // error with error code
		log.Printf("Error occured. Errcode: %d", Errcode)
		errCodeStr := strconv.Itoa(Errcode)
		de = dataerr{Errcode: errCodeStr, Errmessage: Errormessage}
		w.WriteHeader(Errcode)
		err := Templates.ExecuteTemplate(w, "error.html", de)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}
	de = dataerr{Errcode: "", Errmessage: Errormessage} // error without error code
	log.Printf("Error occured. Errcode: %d, ErrMessage: %s", Errcode, Errormessage)
	err := Templates.ExecuteTemplate(w, "error.html", de)
	if err != nil {
		log.Fatal(err.Error())
	}
	return

}

// CutContent cuts long content for preview
func CutContent(posts []postsmanager.Post) []postsmanager.Post {
	var cutted []postsmanager.Post
	var cutter []rune
	for _, el := range posts {
		cutter = []rune(el.Content)
		if len(cutter) > 90 {
			el.Content = el.Content[:100] + "..."
			cutted = append(cutted, el)
		} else {
			cutted = append(cutted, el)
		}
	}
	return cutted
}

// ValidateUnameAndPass checks if username and pass are alphanumeric
func ValidateUnameAndPass(username, pass string) bool {
	valid := false
	for _, letter := range username {
		if letter != ' ' {
			valid = true
		}
	}
	for _, letter := range pass {
		if letter != ' ' {
			valid = true
		}
	}
	return valid
}

//ThreadsCount count id of threads
func ThreadsCount(threadsIDs []string) int {
	count := 0
	for range threadsIDs {
		count++
	}
	return count
}

// IsEmpty checks if title or content is empty or spaced
func IsEmpty(str string) bool {
	return strings.Trim(str, " ") == ""
}
