package auth
import "testing"
func TestPasswordHashAndCheck(t *testing.T){h,err:=HashPassword("correct horse battery staple");if err!=nil{t.Fatal(err)};if CheckPassword(h,"correct horse battery staple")!=nil{t.Fatal("expected match")};if CheckPassword(h,"wrong")!=ErrInvalidCredentials{t.Fatal("expected invalid credentials")}}
func TestPasswordLength(t *testing.T){if _,err:=HashPassword("short");err==nil{t.Fatal("expected short rejection")};if _,err:=HashPassword(string(make([]byte,73)));err==nil{t.Fatal("expected long rejection")}}
