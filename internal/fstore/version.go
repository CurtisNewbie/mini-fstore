package fstore

import "github.com/curtisnewbie/miso/miso"

const (
	Version = "v0.1.11"
)

func init() {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		rail.Infof("mini-fstore version: %v", Version)
		return nil
	})
}
