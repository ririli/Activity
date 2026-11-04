package main
import("crypto/ed25519";"crypto/rand";"encoding/hex";"os";"path/filepath")
func main(){dir:=".local/keys";if e:=os.MkdirAll(dir,0700);e!=nil{panic(e)};for _,k:=range []string{"access","refresh"}{pub,private,e:=ed25519.GenerateKey(rand.Reader);if e!=nil{panic(e)};for name,b:=range map[string][]byte{k+".seed":private.Seed(),k+".pub":pub}{f,e:=os.OpenFile(filepath.Join(dir,name),os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if os.IsExist(e){continue};if e!=nil{panic(e)};_,e=f.WriteString(hex.EncodeToString(b));f.Close();if e!=nil{panic(e)}}}}
