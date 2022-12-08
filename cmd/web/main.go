package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Random-7/GoRcon/pkg/config"
	"github.com/Random-7/GoRcon/pkg/database"
	"github.com/Random-7/GoRcon/pkg/handlers"
	"github.com/Random-7/GoRcon/pkg/rcon"
	"github.com/Random-7/GoRcon/pkg/render"
	"github.com/alexedwards/scs/v2"
	"github.com/spf13/viper"
)

const portNumber = ":8080"

// Application state
var app config.AppConfig

func main() {

	// Setup Viper + defaults
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	viper.SetDefault("portNumber", "8080")
	viper.SetDefault("InProduction", false)
	viper.SetDefault("Version", "0.1alpha")

	//InProduction
	app.InProduction = false
	app.Version = "0.1alpha"

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
	}

	//Setup session and store in appconfig
	app.Session = scs.New()
	app.Session.Lifetime = 24 * time.Hour
	app.Session.Cookie.Persist = true
	app.Session.Cookie.SameSite = http.SameSiteLaxMode
	app.Session.Cookie.Secure = viper.GetBool("InProduction")
	app.TemplateCache = tc //Pass the AppConfig to the render so it can update the template cache
	app.UseCache = viper.GetBool("useCache")

	//Create new repo and pass it back to handlers
	repo := handlers.NewRepo(&app)
	handlers.NewHandlers(repo)
	render.NewTemplates(&app)

	// Establish rcon connection
	rconIP := viper.GetString("RconIP")
	rconPass := viper.GetString("RconPass")
	go setupRconConnection(rconIP, rconPass)

	// Establish database connection
	dbIP := viper.GetString("dbIP")
	dbPort := viper.GetString("dbPort")
	dbUser := viper.GetString("dbUser")
	dbPass := viper.GetString("dbPass")
	dbName := viper.GetString("dbName")
	go setupDatabaseConnection(dbIP, dbPort, dbUser, dbPass, dbName)

	//
	fmt.Println("Starting Webserver on", portNumber)
	srv := &http.Server{
		Addr:    portNumber,
		Handler: setupRouter(&app),
	}
	_ = srv.ListenAndServe()
}

// TODO add checks in this to make sure we have valid info.
// setupDatabaseConnection populates the dbSession struct with info needed to connect to the database and calls the initial connection
func setupDatabaseConnection(ip, port, user, password, dbname string) {
	dbSession := new(database.Session)
	dbSession.IP = ip
	dbSession.Port = port
	dbSession.User = user
	dbSession.Password = password
	dbSession.DbName = dbname
	app.DbSession = *dbSession

	fmt.Println(dbSession.DbName)
	fmt.Println(dbSession.Password)
	fmt.Println(dbSession.User)
	fmt.Println(dbSession.IP)
	fmt.Println(dbSession.Port)

	app.DbSession.Setup()

}

// setupRconConnection buils the rcon connection and saves it in the app config
func setupRconConnection(ip string, password string) {
	//Setup rcon conneciton
	rcon := new(rcon.Connection)
	rcon.Ip = ip
	rcon.Password = password
	//pass into appconfig
	app.Rcon = *rcon
	err := app.Rcon.SetupConnection()
	if err != nil {
		app.Rcon.ConnectionStatus = false
		fmt.Println(err)
	}

}
