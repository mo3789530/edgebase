package main

import (
	"log"
	"net"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/edgebase/platform/kvm-manager/internal/handler"
	"github.com/edgebase/platform/kvm-manager/internal/service"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Try to connect to libvirt
	var l *libvirt.Libvirt
	c, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 2*time.Second)
	if err != nil {
		log.Printf("Warning: failed to dial libvirt: %v. Running in mock mode.", err)
	} else {
		l = libvirt.New(c)
		if err := l.Connect(); err != nil {
			log.Printf("Warning: failed to connect to libvirt: %v. Running in mock mode.", err)
			l = nil
		} else {
			log.Println("Connected to libvirt")
			defer func() {
				if err := l.Disconnect(); err != nil {
					log.Printf("failed to disconnect: %v", err)
				}
			}()
		}
	}

	vmManager := service.NewKVMManager(l)
	h := handler.NewHandler(vmManager)

	app := fiber.New()

	api := app.Group("/api/v1")
	api.Post("/vms", h.CreateVM)
	api.Get("/vms", h.ListVMs)
	api.Get("/vms/:id", h.GetVM)
	api.Delete("/vms/:id", h.DeleteVM)
	api.Post("/vms/:id/start", h.StartVM)
	api.Post("/vms/:id/stop", h.StopVM)

	log.Println("Starting KVM Manager on :8081")
	log.Fatal(app.Listen(":8081"))
}
