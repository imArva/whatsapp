package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"

	// "go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client

func main() {
	app := fiber.New()

	app.Get("/send/:no", sendOTP)

	dbLog := waLog.Stdout("Database", "error", false)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "error", false)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	app.Listen(":8080")
	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func sendOTP(c *fiber.Ctx) error {
	credential := c.Query("credential")
	if credential != "alanarvafian" {
		return c.SendString("Need Valid Credentials")
	}
	noTelp := c.Params("no")
	noTelpJID, _ := parseJID(noTelp)
	min := 100000
	max := 999999
	kode := (rand.Intn(max-min+1) + 1)
	client.SendMessage(context.Background(), noTelpJID, "", &waProto.Message{
		Conversation: proto.String("Kode OTP Dochere mu: " + strconv.Itoa(kode)),
	})

	return c.JSON(fiber.Map{
		"NoTelp": noTelp,
		"Kode":   kode,
	})
}

// func eventHandler(evt interface{}) {
// 	switch v := evt.(type) {
// 	case *events.Message:
//         v.Message.Chat

// 	}
// }

func parseJID(arg string) (types.JID, bool) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			fmt.Errorf("Invalid JID %s: %v", arg, err)
			return recipient, false
		} else if recipient.User == "" {
			fmt.Errorf("Invalid JID %s: no server specified", arg)
			return recipient, false
		}
		return recipient, true
	}
}
