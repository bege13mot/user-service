package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	pb "github.com/bege13mot/user-service/proto/auth"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

const (
	defaultGrpcAddr     = "localhost:50054"
	defaultGrpcHTTPAddr = "10.0.0.65:8081"
	defaultConsulAddr   = "localhost:8500"
)

var (
	// Get database details from environment variables
	dbHost     = os.Getenv("DB_HOST")
	dbUser     = os.Getenv("DB_USER")
	dbName     = os.Getenv("DB_NAME")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbPort     = os.Getenv("DB_PORT")

	grpcAddr     = os.Getenv("GRPC_ADDR")
	grpcHTTPAddr = os.Getenv("GRPC_HTTP_ADDR")
	consulAddr   = os.Getenv("CONSUL_ADDR")
)

func initVar() {
	if dbHost == "" && dbName == "" {
		log.Println("Use default DB connection settings")

		dbHost = "localhost"
		pg := "postgres"
		dbUser = pg
		dbName = pg
		dbPassword = pg
		dbPort = "5433"
	}

	if grpcAddr == "" && grpcHTTPAddr == "" {
		log.Println("Use default GRPC connection settings")
		grpcAddr = defaultGrpcAddr
		grpcHTTPAddr = defaultGrpcHTTPAddr
	}

	if consulAddr == "" {
		log.Println("Use default Consul connection settings")
		consulAddr = defaultConsulAddr
	}
}

// allowCORS allows Cross Origin Resoruce Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	log.Printf("CORS preflight request for %s \n", r.URL.Path)
}

func main() {
	// Creates a database connection and handles
	// closing it again before exit.
	initVar()
	db := CreateConnection()
	defer db.Close()

	// Automatically migrates the user struct
	// into database columns/types etc. This will
	// check for changes and migrate them each time
	// this service is restarted.
	db.AutoMigrate(&pb.User{})

	repo := &userRepository{db}
	tokenService := &tokenService{repo}

	//Connect to Consul
	config := consulapi.DefaultConfig()
	config.Address = consulAddr
	consul, err := consulapi.NewClient(config)
	if err != nil {
		log.Println("Error during connect to Consul, ", err)
	}

	serviceID := "user-service_" + grpcAddr

	//Register in Consul
	defer func() {
		cErr := consul.Agent().ServiceDeregister(serviceID)
		if cErr != nil {
			log.Println("Cant add service to Consul", cErr)
			return
		}
		log.Println("Deregistered in Consul", serviceID)
	}()

	err = consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "user-service",
		Port:    50054,
		Address: "host123",
		Check: &consulapi.AgentServiceCheck{
			CheckID:  "health_check",
			Name:     "User-Service health status",
			Interval: "10s",
			GRPC:     "host123:50054",
		},
	})
	if err != nil {
		log.Println("Couldn't register service in Consul, ", err)
	}
	log.Println("Registered in Consul", serviceID)

	//Test section
	health, _, err := consul.Health().Service("user-service", "", false, nil)
	if err != nil {
		log.Println("Cant get alive services")
	}

	fmt.Println("HEALTH: ", len(health))
	for _, item := range health {
		fmt.Println("Checks: ", item.Checks, item.Checks.AggregatedStatus())
		fmt.Println("Service: ", item.Service.ID, item.Service.Address, item.Service.Port)
		fmt.Println("--- ")
	}

	//Get IP
	ip, err := net.InterfaceAddrs()
	if err != nil {
		log.Println("Couldn't get IP address")
	}
	for _, a := range ip {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("MY IP: ", ipnet.IP)
			}
		}
	}

	// fire the gRPC server in a goroutine anonymous function
	go func() {
		// create a listener on TCP port
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("Failed to listen port %v, error: %v", grpcAddr, err)
		}

		// create a gRPC server object
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		)

		pb.RegisterAuthServer(grpcServer, &service{repo, tokenService})
		pb.RegisterHealthServer(grpcServer, &service{repo, tokenService})

		// Initialize all metrics.
		grpc_prometheus.Register(grpcServer)

		// start the server
		log.Printf("starting gRPC server on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %s", err)
		}
	}()

	// fire the REST server in a goroutine anonymous function
	go func() {

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()

		// Setup the client gRPC options
		opts := []grpc.DialOption{grpc.WithInsecure()}

		// Register Gateway
		err := pb.RegisterAuthHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
		if err != nil {
			log.Fatalf("Failed to register AuthHandler: %s, port: %s", err, grpcAddr)
		}

		httpMux := http.NewServeMux()
		httpMux.Handle("/", mux)
		httpMux.Handle("/metrics", promhttp.Handler())
		httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if err := db.DB().Ping(); err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		httpMux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
			dir := "./proto/auth"
			if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
				log.Printf("Swagger Not Found: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			log.Printf("Serving Swagger %s", r.URL.Path)
			p := strings.TrimPrefix(r.URL.Path, "/swagger/")
			p = path.Join(dir, p)
			fmt.Println(p)
			http.ServeFile(w, r, p)
		})

		s := &http.Server{
			Addr:    grpcHTTPAddr,
			Handler: allowCORS(httpMux),
		}

		log.Printf("Starting REST server on %s", grpcHTTPAddr)
		s.ListenAndServe()
	}()

	// infinite loop
	select {}
}
