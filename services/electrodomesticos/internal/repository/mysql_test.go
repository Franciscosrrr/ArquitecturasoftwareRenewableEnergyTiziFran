package repository
import("context";"fmt";"os";"testing";"time")
func TestMySQLIntegration(t *testing.T){
 password:=os.Getenv("MYSQL_TEST_PASSWORD");if password==""{t.Skip("requiere MySQL real: perfil verificar-go")};s,err:=Open("catalogo",password,os.Getenv("MYSQL_TEST_ADDR"),"electrodomesticos_db");if err!=nil{t.Fatal(err)};defer s.DB.Close();ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 if err=s.Migrate(ctx);err!=nil{t.Fatal(err)};first,err:=s.Resolve(ctx,[]string{"heladera","iluminacion","lavarropas"});if err!=nil||len(first)!=3{t.Fatalf("seed: %v %v",first,err)};if err=s.Migrate(ctx);err!=nil{t.Fatal(err)};again,err:=s.Resolve(ctx,[]string{"heladera","iluminacion","lavarropas"});if err!=nil||len(again)!=3{t.Fatal("seed no idempotente")}
 id:=fmt.Sprintf("integration-%d",time.Now().UnixNano());_,err=s.DB.ExecContext(ctx,`INSERT INTO electrodomesticos(id,nombre,categoria,modo,potencia_w,fuente) VALUES(?,?,'Prueba','POTENCIA',10,'fixture de integración')`,id,"Prueba literal %_");if err!=nil{t.Fatal(err)};defer s.DB.ExecContext(context.Background(),"DELETE FROM electrodomesticos WHERE id=?",id)
 p,err:=s.List(ctx,Query{Text:"%_",Category:"Prueba",Page:1,Size:1,Sort:"nombre"});if err!=nil||p.Total!=1||len(p.Items)!=1||p.Items[0].ID!=id{t.Fatalf("wildcard/page: %+v %v",p,err)}
 _,err=s.DB.ExecContext(ctx,"UPDATE electrodomesticos SET activo=FALSE WHERE id=?",id);if err!=nil{t.Fatal(err)};items,err:=s.Resolve(ctx,[]string{id});if err!=nil||len(items)!=0{t.Fatal("inactive item exposed")}
 p,err=s.List(ctx,Query{Text:"' OR 1=1 --",Page:1,Size:20});if err!=nil||p.Total!=0{t.Fatal("query must be parameterized")}
}
