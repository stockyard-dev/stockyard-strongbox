package main
import ("fmt";"log";"net/http";"os"
"github.com/stockyard-dev/stockyard-strongbox/internal/server"
"github.com/stockyard-dev/stockyard-strongbox/internal/store")
func main() {
port:=os.Getenv("PORT");if port==""{port="8610"}
dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./strongbox-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("strongbox: %v",err)};defer db.Close()
srv:=server.New(db)
fmt.Printf("\n  Strongbox — Self-hosted secret manager\n  ─────────────────────────────────\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Data:       %s\n  ─────────────────────────────────\n\n",port,port,dataDir)
log.Printf("strongbox: listening on :%s",port)
if err:=http.ListenAndServe(":"+port,srv);err!=nil{log.Fatalf("strongbox: %v",err)}}
