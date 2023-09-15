// Модуль rpcsrv представляет абстрацию сервера по обработке апросов по протоколу gRPC.
package rpcsrv

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// импортируем пакет со сгенерированными protobuf-файлами
	cookie "github.com/jon69/shorturl/internal/app/cookie"
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
func MakeServer(k []byte, baseURL string, conndb string, urlstorage *storage.StorageURL) *PRCServer {
	srv := &PRCServer{}

	mygrpcsrv := &gPRCServer{}
	mygrpcsrv.conndb = conndb
	mygrpcsrv.urlstorage = urlstorage
	mygrpcsrv.baseURL = baseURL
	mygrpcsrv.key = k
	// 	создаем сервис
	srv.grpcserver = grpc.NewServer(grpc.UnaryInterceptor(mygrpcsrv.shorturlInterceptor))

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
	// key - секретный ключ на подписи куки.
	key []byte
	// baseURL - адрес (хост:порт) для выдачи сохраненных URL.
	baseURL string
	// conndb - параметры подключения к БД.
	conndb string
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedShortURLServer
	// urlstorage - хранилище данных.
	urlstorage *storage.StorageURL
}

// CTXUid структура для хранения конекста запроса с информацией о польльзователе.
type CTXUid struct {
}

func (h *gPRCServer) shorturlInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Print("shorturlInterceptor called")

	var cookieName string
	var cookieValue string

	var newCookieName string
	var newCookieValue string

	var uid string

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("cookie_name")
		if len(values) > 0 {
			cookieName = values[0]
		}
		values = md.Get("cookie_value")
		if len(values) > 0 {
			cookieValue = values[0]
		}
	}

	if len(cookieValue) == 0 || len(cookieName) == 0 {
		log.Println("cookie empty")
		newCookieName, newCookieValue, uid = cookie.GetNewSignedCookie(h.key)
	} else {
		var equal bool
		equal, uid = cookie.ValidateCookie(h.key, cookieName, cookieValue)
		if !equal {
			newCookieName, newCookieValue, uid = cookie.GetNewSignedCookie(h.key)
		} else {
			newCookieName = cookieName
			newCookieValue = cookieValue
		}
	}

	header := metadata.New(map[string]string{"cookie_name": newCookieName, "cookie_value": newCookieValue})
	if err := grpc.SendHeader(ctx, header); err != nil {
		log.Println("gPRCServer can not send cookie: " + err.Error())
		return nil, status.Errorf(codes.Internal, "unable to send cookie")
	}

	ctx2 := context.WithValue(ctx, CTXUid{}, uid)
	return handler(ctx2, req)
}

// Ping обрабатывает запрос на проверку подключения к БД
func (h *gPRCServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	log.Println("gPRCServer Ping")

	var response pb.PingResponse
	response.Stmsg = &pb.StatusMessage{Status: pb.StatusMessage_OK}

	if h.conndb != "" {
		if !dbh.Ping(h.conndb) {
			response.Stmsg.Status = pb.StatusMessage_ERROR
		}
	}
	log.Println("return from gPRCServer Ping")
	return &response, nil
}

// PostURL обрабатывает запрос на создание укороченной ссылки URL
func (h *gPRCServer) PostURL(ctx context.Context, in *pb.PostURLRequest) (*pb.PostURLResponse, error) {
	log.Print("gPRCServer PostURL url=" + in.Url)

	var response pb.PostURLResponse
	response.Stmsg = &pb.StatusMessage{Status: pb.StatusMessage_OK}

	var id string
	var iou int
	iou, id = h.urlstorage.PutURL(in.Url)

	if iou != 1 {
		response.Stmsg.Status = pb.StatusMessage_ERROR
	}

	response.ShortUrl = id
	return &response, nil
}

// GetURL обрабатывает запрос на получение URL
func (h *gPRCServer) GetURL(ctx context.Context, in *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	log.Print("gPRCServer GetURL id=" + in.Id)

	var response pb.GetURLResponse
	response.Stmsg = &pb.StatusMessage{Status: pb.StatusMessage_OK}
	val, ok, isDel := h.urlstorage.GetURL(in.Id)

	if ok {
		if isDel {
			response.Stmsg.Status = pb.StatusMessage_NOT_FOUND
		} else {
			response.Url = val
		}
	} else {
		response.Stmsg.Status = pb.StatusMessage_NOT_FOUND
	}

	return &response, nil
}
