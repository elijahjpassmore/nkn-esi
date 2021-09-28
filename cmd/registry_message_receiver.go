package cmd

import (
	"github.com/elijahjpassmore/nkn-esi/api/esi"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

// registryMessageReceiver receives and returns any incoming registry messages.
func registryMessageReceiver() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	<-registryClient.OnConnect.C
	log.WithFields(log.Fields{
		"publicKey": registryInfo.GetPublicKey(),
		"name": registryInfo.GetName(),
	}).Info("Connection opened")

	message := &esi.RegistryMessage{}

	for {
		msg := <-registryClient.OnMessage.C

		log.WithFields(log.Fields{
			"src": msg.Src,
		}).Info("Message received")

		err := proto.Unmarshal(msg.Data, message)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		// Case documentation located at api/esi/der_facility_registry_service.go.
		switch x := message.Chunk.(type) {
		case *esi.RegistryMessage_SignupRegistry:
			if _, ok := knownFacilities[x.SignupRegistry.PublicKey]; !ok {
				log.WithFields(log.Fields{
					"publicKey": msg.Src,
				}).Info("Saved facility to known facilities")

				for _, facility := range knownFacilities {
					err = esi.SendKnownDerFacility(registryClient, msg.Src, facility)
					if err != nil {
						log.Error(err.Error())
					}
				}

				knownFacilities[x.SignupRegistry.PublicKey] = x.SignupRegistry
			}

		case *esi.RegistryMessage_QueryDerFacilities:
			// TODO: Look at more than just country.
			log.WithFields(log.Fields{
				"publicKey": msg.Src,
			}).Info("Query for facility")

			for _, facility := range knownFacilities {
				if strings.ToLower(facility.Location.GetCountry()) == strings.ToLower(x.QueryDerFacilities.Location.GetCountry()) {

					// If the facility querying the registry also fits the criteria, ignore it.
					if facility.PublicKey == msg.Src {
						continue
					}

					err = esi.SendKnownDerFacility(registryClient, msg.Src, facility)
					if err != nil {
						log.Error(err.Error())
					}

					log.WithFields(log.Fields{
						"end": msg.Src,
					}).Info("Sent known facility")
				}
			}
		}
	}
}
