package client

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/0xcafed00d/joystick"
	"github.com/Speshl/GoRemoteControl_Client/client/controllers"
	"github.com/Speshl/GoRemoteControl_Server/models"
)

type Client struct {
	address        string
	cfgPath        string
	invertEsc      bool
	invertSteering bool
	trimSteering   int
}

func NewClient(address string, cfgPath string, invertEsc bool, invertSteering bool, trimSteering int) *Client {
	log.Printf("Server Address: %s, CFG Path: %s, InvertESC: %t, InvertSteering: %t, TrimSteering: %d", address, cfgPath, invertEsc, invertSteering, trimSteering)
	client := Client{
		address:        address,
		cfgPath:        cfgPath,
		invertEsc:      invertEsc,
		invertSteering: invertSteering,
		trimSteering:   trimSteering,
	}
	return &client
}

func (c *Client) RunClient(ctx context.Context) error {
	log.Println("starting client...")
	defer log.Println("client stopped")

	conn, err := net.Dial("udp", c.address) //TODO: IP as a param
	if err != nil {
		return err
	}
	defer conn.Close()

	joySticks, err := GetJoysticks()
	if err != nil {
		return err
	}
	defer func() {
		for _, js := range joySticks {
			js.Close()
		}
	}()

	controller, err := controllers.CreateController(joySticks, c.cfgPath, c.invertEsc, c.invertSteering, c.trimSteering)
	if err != nil {
		return err
	}
	log.Println("start sending...")
	defer log.Println("sending stopped")
	ticker := time.NewTicker(4 * time.Millisecond)
	var lastSent models.Packet
	logTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-logTicker.C:
			log.Printf("ticker log - latest UDP packet sent: %+v\n", lastSent.State)
		case <-ticker.C:
			state, err := controller.GetUpdatedState()
			if err != nil {
				return fmt.Errorf("failed getting controller state- %w", err)
			}

			statePacket := models.Packet{
				StateType: state.GetType(),
				State:     state,
				SentAt:    time.Now(),
			}
			var buffer bytes.Buffer
			encoder := gob.NewEncoder(&buffer)
			gob.Register(models.GroundState{})
			err = encoder.Encode(statePacket)
			if err != nil {
				return err
			}
			_, err = conn.Write(buffer.Bytes())
			if err != nil {
				return err
			}
			//log.Printf("%+v\n", statePacket.State)
			lastSent = statePacket
		}
	}
}

func GetJoysticks() ([]joystick.Joystick, error) {
	joySticks := make([]joystick.Joystick, 0)

	for i := 0; i < 10; i++ {
		js, err := joystick.Open(i)
		if err != nil {
			if i == 0 {
				return nil, fmt.Errorf("no joysticks found - %w\n", err)
			}
			break //not an issue if we got atleast 1
		}
		log.Printf("Joystick Name: %s", js.Name())
		log.Printf("   Axis Count: %d", js.AxisCount())
		log.Printf(" Button Count: %d", js.ButtonCount())
		joySticks = append(joySticks, js)
	}
	return joySticks, nil
}

func ShowJoyStats() ([]joystick.Joystick, error) {
	joySticks, err := GetJoysticks()
	if err != nil {
		return nil, err
	}

	for {
		for i, joystick := range joySticks {
			state, err := joystick.Read()
			if err != nil {
				return nil, err
			}
			log.Printf("Joystick %d state: %+v\n", i, state)

		}
		time.Sleep(1000)
	}
}
