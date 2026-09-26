package auth
import("testing";"time";"github.com/google/uuid")
func TestTokenRoundTrip(t *testing.T){now:=time.Unix(1700000000,0);uid:=uuid.New();sid:=uuid.New();m,err:=NewTokenManager("01234567890123456789012345678901","web-studio-img",time.Hour);if err!=nil{t.Fatal(err)};token,_,err:=m.Issue(uid,sid,now);if err!=nil{t.Fatal(err)};claims,err:=m.Parse(token,now.Add(time.Minute));if err!=nil{t.Fatal(err)};if claims.UserID!=uid||claims.SessionID!=sid{t.Fatal("claims mismatch")}}
