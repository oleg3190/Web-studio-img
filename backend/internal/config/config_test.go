package config
import "testing"
func TestLoadDefaults(t *testing.T){t.Setenv("APP_ENV","development");t.Setenv("APP_PORT","");t.Setenv("JWT_SECRET","");c,e:=Load();if e!=nil{t.Fatal(e)};if c.Port!="8080"{t.Fatalf("port=%q",c.Port)}}
func TestProductionSecret(t *testing.T){t.Setenv("APP_ENV","production");t.Setenv("APP_PORT","8080");t.Setenv("JWT_SECRET","");if _,e:=Load();e==nil{t.Fatal("expected error")}}
func TestInvalidPort(t *testing.T){t.Setenv("APP_ENV","development");t.Setenv("APP_PORT","abc");if _,e:=Load();e==nil{t.Fatal("expected error")}}
