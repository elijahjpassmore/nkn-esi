package cmd

import (
	"fmt"
	"github.com/elijahjpassmore/nkn-esi/api/esi"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// coordinationNodeMessageReceiver receives and returns any incoming coordination node messages.
func coordinationNodeMessageReceiver() {
	var formKey int // a simple number to increment form number.

	<-coordinationNodeClient.OnConnect.C
	log.WithFields(log.Fields{
		"publicKey": coordinationNodeInfo.GetPublicKey(),
		"name":      coordinationNodeInfo.GetName(),
	}).Info("Connection opened")

	message := &esi.FacilityMessage{}

	for {
		// Unmarshal the protocol buffer.
		msg := <-coordinationNodeClient.OnMessage.C
		err := proto.Unmarshal(msg.Data, message)
		if err != nil {
			log.Error(err.Error())
		}

		// Case documentation located at api/esi/der_facility_service.go.
		//
		// Switch based upon the message type.
		switch x := message.Chunk.(type) {
		case *esi.FacilityMessage_SendKnownDerFacility:
			// If the node is not already stored, store it.
			_, present := knownCoordinationNodes[x.SendKnownDerFacility.GetPublicKey()]
			if !present {
				knownCoordinationNodes[x.SendKnownDerFacility.PublicKey] = x.SendKnownDerFacility
			}

			log.WithFields(log.Fields{
				"src": msg.Src,
			}).Info(fmt.Sprintf("Saved coordination node %s", x.SendKnownDerFacility.GetPublicKey()))

		case *esi.FacilityMessage_GetDerFacilityRegistrationForm:
			// Set the basic info.
			//
			// An example FormSetting - you can set whatever you want, and the facility will get a copy for you to then
			// evaluate as you wish.
			newFormSetting := esi.FormSetting{
				Key:         "0",
				Label:       "Do you wish to register?",
				Caption:     "",
				Placeholder: "Y",
			}
			newForm := esi.Form{
				LanguageCode: "en",
				Key:          strconv.Itoa(formKey),
				Settings:     []*esi.FormSetting{&newFormSetting},
			}
			newRoute := esi.DerRoute{
				FacilityKey: msg.Src,
				ExchangeKey: coordinationNodeInfo.GetPublicKey(),
			}
			newRegistrationForm := esi.DerFacilityRegistrationForm{
				Route: &newRoute,
				Form:  &newForm,
			}

			// Send the registration form.
			err = esi.SendDerFacilityRegistrationForm(coordinationNodeClient, &newRegistrationForm)
			if err != nil {
				log.Error(err.Error())
			}

			formKey += 1 // increment form key

			log.WithFields(log.Fields{
				"end": msg.Src,
			}).Info("Sent registration form")

		case *esi.FacilityMessage_SendDerFacilityRegistrationForm:
			log.WithFields(log.Fields{
				"src": msg.Src,
			}).Info("Received registration form")

			// If the form is not already stored, store it.
			_, present := receivedRegistrationForms[x.SendDerFacilityRegistrationForm.Route.GetExchangeKey()]
			if !present {
				receivedRegistrationForms[x.SendDerFacilityRegistrationForm.Route.GetExchangeKey()] = x.SendDerFacilityRegistrationForm
			}

		case *esi.FacilityMessage_SubmitDerFacilityRegistrationForm:
			log.WithFields(log.Fields{
				"src": msg.Src,
			}).Info("Received registration form data")

			registration := esi.DerFacilityRegistration{
				Route: x.SubmitDerFacilityRegistrationForm.Route,
			}
			// If the user responded positively, then success.
			response := strings.ToLower(x.SubmitDerFacilityRegistrationForm.Data.Data["0"])
			if response == "y" || response == "yes" {
				registration.Success = true
			} else {
				registration.Success = false
			}

			// If successful, add it as a facility.
			if registration.Success {
				registeredFacilities[msg.Src] = true
			}

			err = esi.CompleteDerFacilityRegistration(coordinationNodeClient, &registration)
			if err != nil {
				log.Error(err.Error())
			}

			log.WithFields(log.Fields{
				"end":     msg.Src,
				"success": registration.GetSuccess(),
			}).Info("Sent completed registration form")

		case *esi.FacilityMessage_CompleteDerFacilityRegistration:
			if x.CompleteDerFacilityRegistration.GetSuccess() == true {
				registeredExchange = msg.Src
			}
			log.WithFields(log.Fields{
				"src":     msg.Src,
				"success": x.CompleteDerFacilityRegistration.GetSuccess(),
			}).Info("Received completed registration form")

		case *esi.FacilityMessage_GetResourceCharacteristics:
			// Check to make sure that the source is the registered exchange.
			if registeredExchange == msg.Src {
				newRoute := esi.DerRoute{
					FacilityKey: coordinationNodeInfo.GetPublicKey(),
					ExchangeKey: msg.Src,
				}
				// TODO: fix
				newCharacteristics := resourceCharacteristics
				newCharacteristics.Route = &newRoute
				err := esi.SendResourceCharacteristics(coordinationNodeClient, &newCharacteristics)
				if err != nil {
					log.Error(err.Error())
				}

				log.WithFields(log.Fields{
					"end": msg.Src,
				}).Info("Sent resource characteristics")
			}

		case *esi.FacilityMessage_SendResourceCharacteristics:
			// Check to make sure that the source is a registered facility.
			if registeredFacilities[msg.Src] == true {
				facilityCharacteristics[msg.Src] = x.SendResourceCharacteristics

				log.WithFields(log.Fields{
					"src": msg.Src,
				}).Info("Received resource characteristics")
			}

		case *esi.FacilityMessage_GetPriceMap:
			// Check to make sure that the source is the registered exchange.
			if registeredExchange == msg.Src {
				err = esi.SendPriceMap(coordinationNodeClient, x.GetPriceMap.Route.GetExchangeKey(), &priceMap)
				if err != nil {
					log.Error(err.Error())
				}

				log.WithFields(log.Fields{
					"end": msg.Src,
				}).Info("Sent price map")
			}

		case *esi.FacilityMessage_SendPriceMap:
			// Check to make sure that the source is a registered facility.
			if registeredFacilities[msg.Src] == true {
				facilityPriceMaps[msg.Src] = x.SendPriceMap

				log.WithFields(log.Fields{
					"src": msg.Src,
				}).Info("Received price map")
			}

		case *esi.FacilityMessage_ProposePriceMapOffer:
			// Check to make sure that the source is the registered exchange.
			if registeredExchange == msg.Src || registeredFacilities[msg.Src] == true {
				log.Info("RECEIVED PROPOSE OFFER")
				if x.ProposePriceMapOffer.PriceMap.Price.ApparentEnergyPrice.Units < autoPrice.AlwaysBuyBelowPrice.Units {
					// If the offer is below our auto accept, just accept the offer.
					//
					// There is also a value for "AvoidBuyOverPrice", which could be used in a similar way in other
					// scenarios. In this demo, if the price is not lower than our auto accept, then it just goes to
					// evaluation.
					response := acceptOffer(x.ProposePriceMapOffer.Route, x.ProposePriceMapOffer.OfferId)
					err = esi.SendPriceMapOfferResponse(coordinationNodeClient, response)
					if err != nil {
						log.Error(err.Error())
					}

					log.WithFields(log.Fields{
						"src":  msg.Src,
						"auto": autoPrice.AlwaysBuyBelowPrice.Units,
					}).Info("Accepted price map due to auto buy")

					priceMapOffers[x.ProposePriceMapOffer.OfferId.Uuid] = x.ProposePriceMapOffer
					// Store the status of the offer.
					status := esi.PriceMapOfferStatus{
						Route:   x.ProposePriceMapOffer.Route,
						OfferId: x.ProposePriceMapOffer.OfferId,
						Status:  esi.PriceMapOfferStatus_ACCEPTED,
					}
					priceMapOfferStatus[x.ProposePriceMapOffer.OfferId.Uuid] = &status
				} else {
					priceMapOffers[x.ProposePriceMapOffer.OfferId.Uuid] = x.ProposePriceMapOffer

					log.WithFields(log.Fields{
						"src": msg.Src,
					}).Info("Received price map offer")

					// Store the status of the offer.
					status := esi.PriceMapOfferStatus{
						Route:   x.ProposePriceMapOffer.Route,
						OfferId: x.ProposePriceMapOffer.OfferId,
						Status:  esi.PriceMapOfferStatus_UNKNOWN,
					}
					priceMapOfferStatus[x.ProposePriceMapOffer.OfferId.Uuid] = &status
				}
			}

		case *esi.FacilityMessage_SendPriceMapOfferResponse:
			switch y := x.SendPriceMapOfferResponse.AcceptOneof.(type) {
			// Evaluate the contents of the response.
			case *esi.PriceMapOfferResponse_Accept:
				if y.Accept {
					// If the offer has been accepted, log the acceptance.
					log.WithFields(log.Fields{
						"src": msg.Src,
					}).Info("Price map accepted")

					// Store the status ACCEPTED.
					priceMapOfferStatus[x.SendPriceMapOfferResponse.OfferId.Uuid].Status = esi.PriceMapOfferStatus_ACCEPTED
				}
			case *esi.PriceMapOfferResponse_CounterOffer:
				log.WithFields(log.Fields{
					"src": msg.Src,
				}).Info("Counter offer received")

				// Store the previous offer as REJECTED.
				priceMapOfferStatus[x.SendPriceMapOfferResponse.PreviousOffer.Uuid].Status = esi.PriceMapOfferStatus_REJECTED

				// Set the responsible party.
				//
				// In this case, it's just assumed that the responsible party is the opposite of whoever it was before.
				// var party = esi.NodeType_NONE
				// if priceMapOffers[x.SendPriceMapOfferResponse.PreviousOffer.Uuid].Node.Type == esi.NodeType_FACILITY {
				// 	party = esi.NodeType_EXCHANGE
				// } else {
				// 	party = esi.NodeType_FACILITY
				// }

				// In the new offer, use the time specified by the previous offer.
				newOffer := esi.PriceMapOffer{
					Route:    x.SendPriceMapOfferResponse.Route,
					OfferId:  x.SendPriceMapOfferResponse.OfferId,
					When:     priceMapOffers[x.SendPriceMapOfferResponse.PreviousOffer.Uuid].When,
					PriceMap: x.SendPriceMapOfferResponse.GetCounterOffer(),
					//Node:     &esi.NodeType{Type: party},
					Node: x.SendPriceMapOfferResponse.Node,
				}
				// Store the new offer.
				priceMapOffers[x.SendPriceMapOfferResponse.OfferId.Uuid] = &newOffer

				// Store the status of the offer.
				status := esi.PriceMapOfferStatus{
					Route:   x.SendPriceMapOfferResponse.Route,
					OfferId: x.SendPriceMapOfferResponse.OfferId,
					Status:  esi.PriceMapOfferStatus_UNKNOWN,
				}
				priceMapOfferStatus[x.SendPriceMapOfferResponse.OfferId.Uuid] = &status

				if y.CounterOffer.Price.ApparentEnergyPrice.Units < autoPrice.AlwaysBuyBelowPrice.Units {
					// If it falls below the auto accept, then accept it.
					response := acceptOffer(x.SendPriceMapOfferResponse.Route, x.SendPriceMapOfferResponse.OfferId)
					err = esi.SendPriceMapOfferResponse(coordinationNodeClient, response)
					if err != nil {
						log.Error(err.Error())
					}

					priceMapOfferStatus[x.SendPriceMapOfferResponse.OfferId.Uuid].Status = esi.PriceMapOfferStatus_ACCEPTED

					log.WithFields(log.Fields{
						"src":  msg.Src,
						"auto": autoPrice.AlwaysBuyBelowPrice.Units,
					}).Info("Accepted price map due to auto buy")
				}
			}

		case *esi.FacilityMessage_GetPriceMapOfferFeedback:
			// As mentioned in coordination_node_input_receiver.go, this is merely a stub of what could be implemented.
			//
			// In a real situation, getting feedback on a response (either manually or automatically) is very powerful,
			// this is just to show the capability.
			if registeredFacilities[msg.Src] == true {
				log.WithFields(log.Fields{
					"src":   msg.Src,
					"claim": x.GetPriceMapOfferFeedback.ObligationStatus,
				}).Info("Received offer feedback")

				agreement := true
				response := esi.PriceMapOfferFeedbackResponse{
					Route:    x.GetPriceMapOfferFeedback.Route,
					OfferId:  x.GetPriceMapOfferFeedback.OfferId,
					Accepted: agreement,
				}

				err := esi.ProvidePriceMapOfferFeedback(coordinationNodeClient, &response)
				if err != nil {
					log.Error(err.Error())
				}
			}

		case *esi.FacilityMessage_ProvidePriceMapOfferFeedback:
			log.WithFields(log.Fields{
				"src":   msg.Src,
				"claim": x.ProvidePriceMapOfferFeedback.Accepted,
			}).Info("Offer feedback response")
		}
	}
}

func acceptOffer(route *esi.DerRoute, offerId *esi.Uuid) *esi.PriceMapOfferResponse {
	accept := esi.PriceMapOfferResponse_Accept{
		Accept: true,
	}
	response := esi.PriceMapOfferResponse{
		Route:       route,
		OfferId:     offerId,
		AcceptOneof: &accept,
	}

	return &response
}
