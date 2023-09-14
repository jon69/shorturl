// Модуль rpcsrv представляет абстрацию сервера по обработке апросов по протоколу gRPC.
package rpcsrv

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	// импортируем пакет со сгенерированными protobuf-файлами
	dbh "github.com/jon69/shorturl/internal/app/db"
	"github.com/jon69/shorturl/internal/app/storage"
	pb "github.com/jon69/shorturl/proto"
)

// PRCServer представляет RPC сервер.
type PRCServer struct {
	// grpcserver - сервер.
	grpcserver *grpc.Server
}

// MakeServer создает ноый RPC сервер.
func MakeServer(conndb string, urlstorage *storage.StorageURL) *PRCServer {
	srv := &PRCServer{}
	// 	создаем сервис
	srv.grpcserver = grpc.NewServer()
	mygrpcsrv := &gPRCServer{}
	mygrpcsrv.conndb = conndb
	mygrpcsrv.urlstorage = urlstorage
	// регистрируем сервис
	pb.RegisterShortURLServer(srv.grpcserver, mygrpcsrv)
	return srv
}

// Shutdown завершает работу
func (srv *PRCServer) Shutdown() {
	srv.grpcserver.GracefulStop()
}

// Serve запускает сервер на обработку
func (srv *PRCServer) Serve() error {
	// определяем порт для сервера
	listen, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}
	log.Println("Сервер gRPC начал работу")
	// получаем запрос gRPC
	if err := srv.grpcserver.Serve(listen); err != nil {
		return err
	}
	return nil
}

// PRCServer поддерживает все необходимые методы сервера.
type gPRCServer struct {
	// conndb - параметры подключения к БД.
	conndb string
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedShortURLServer
	// urlstorage - хранилище данных.
	urlstorage *storage.StorageURL
}

// Ping обрабатывает запрос на проверку подключения к БД
func (h *gPRCServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	log.Println("gPRCServer Ping")

	var response pb.PingResponse
	response.Ping = pb.PingResponse_OK

	if h.conndb != "" {
		if !dbh.Ping(h.conndb) {
			response.Ping = pb.PingResponse_ERROR
		}
	}
	return &response, nil
}
