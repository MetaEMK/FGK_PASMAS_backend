package pasmasservice

import (
	"errors"
	"time"

	dh "github.com/MetaEMK/FGK_PASMAS_backend/databaseHandler"
	"github.com/MetaEMK/FGK_PASMAS_backend/model"
	"github.com/MetaEMK/FGK_PASMAS_backend/router/realtime"
	"gorm.io/gorm"
)

var (
    ErrNoPilotAvailable = errors.New("No valid pilot available")
    ErrNoStartFuelFound = errors.New("No start fuel found")
    ErrMaxSeatPayload = errors.New("maxSeatPayload was exceeded")
    ErrTooManyPassenger = errors.New("too many passengers for this plane")
    ErrTooLessPassenger = errors.New("A flight needs to have at least one passenger")
    ErrOverloaded = errors.New("MTOW is exceeded")
)

func checkIfSlotIsFree(planeId uint, departureTime time.Time, arrivalTime time.Time) bool {
    var count int64
    result := dh.Db.Model(&model.Flight{}).Where("plane_id = ?", planeId).Where("departure_time < ? AND arrival_time > ?", arrivalTime, departureTime).Count(&count)

    if result.Error != nil {
        return false
    }

    return count == 0
}

func calculatePilot(passWeight uint, fuelAmount float32, plane model.Plane) (model.Pilot, error) {
    var baseETOW uint = 0
    pilot := model.Pilot{}

    err := dh.Db.Preload("AllowedPilots").Preload("PrefPilot").First(&plane).Error
    if err != nil {
        return model.Pilot{}, err
    }

    if plane.PrefPilot == nil {
        if len(*plane.AllowedPilots) > 0 {
            pilot = (*plane.AllowedPilots)[0]
        } else {
            return model.Pilot{}, ErrNoPilotAvailable
        }

    } else {
        pilot = *plane.PrefPilot
    }


    baseETOW += passWeight
    baseETOW += plane.EmptyWeight
    baseETOW += uint(fuelAmount * plane.FuelConversionFactor)

    if plane.MTOW < baseETOW + pilot.Weight {
        newPilot := model.Pilot{}

        if len(*plane.AllowedPilots) == 0 {
            return model.Pilot{}, ErrNoPilotAvailable
        }

        for _, p := range *plane.AllowedPilots {
            if plane.MTOW >= baseETOW + p.Weight {
                newPilot = p
                break
            }
        }

        if newPilot.ID == 0 {
            return model.Pilot{}, ErrOverloaded
        }

        pilot = newPilot
    }

    return pilot, err
}

func checkFlightValidation(flight model.Flight) error {
    var err error
    plane := model.Plane{}
    pilot := model.Pilot{}
    
    planeErr := dh.Db.Preload("Division").First(&plane, flight.PlaneId).Error
    pilotErr := dh.Db.First(&pilot, flight.PilotId).Error

    err = errors.Join(err, planeErr, pilotErr)
    if err != nil {
        return ErrObjectDependencyMissing
    }

    // Validate number of passengers
    if len(*flight.Passengers) > int(plane.Division.PassengerCapacity) {
        return ErrTooManyPassenger
    }

    if len(*flight.Passengers) == 0 {
        return ErrTooLessPassenger
    }

    // Validate if flight is overweight
    var etow float32 = 0
    etow += float32(plane.EmptyWeight)
    etow += *flight.FuelAtDeparture * plane.FuelConversionFactor
    etow += float32(pilot.Weight)

    for _, p := range *flight.Passengers {
        if plane.MaxSeatPayload > 0 {
            if p.Weight > uint(plane.MaxSeatPayload) {
                err = errors.Join(err, ErrMaxSeatPayload)
            }
        }
        etow += float32(p.Weight)
    }

    return err
}


func calculatePassWeight(passengers []model.Passenger, maxSeatPayload int) (uint, error) {
    weight := uint(0)
    for _, p := range passengers {
        if maxSeatPayload > 0 && p.Weight > uint(maxSeatPayload){
            return 0, ErrMaxSeatPayload
        }
        weight += p.Weight
    }

    return weight, nil
}

func calculateFuelAtDeparture(flight *model.Flight, plane model.Plane) (float32, error) {
    if flight.FuelAtDeparture != nil && *flight.FuelAtDeparture != 0 {
        if *flight.FuelAtDeparture > float32(plane.FuelMaxCapacity) {
            return 0, ErrTooMuchFuel
        }
        return *flight.FuelAtDeparture, nil
    }

    // Get one flight before this
    beforeFlight := model.Flight{}
    err := dh.Db.Not("status = ?", model.FsBlocked).Where("plane_id = ?", flight.PlaneId).Where("departure_time < ?", flight.DepartureTime).Order("departure_time DESC").First(&beforeFlight).Error
    if err == gorm.ErrRecordNotFound {
        fuel := float32(plane.FuelStartAmount)
        flight.FuelAtDeparture = &fuel
        return float32(plane.FuelStartAmount), nil
    }

    value, err := calculateFuelAtDeparture(&beforeFlight, plane)
    value -= plane.FuelburnPerFlight

    if value <= 0 {
        return 0, ErrTooLessFuel
    }

    return value, nil
}

func partialUpdatePassengers(db *gorm.DB, oldPass *[]model.Passenger, newPass *[]model.Passenger) {
    if oldPass == nil || newPass == nil {
        return
    }

    if db == nil {
        db = dh.Db.Begin()
    }

    for i := range *newPass {
        switch (*newPass)[i].Action {
        case model.ActionCreate:
            passengerCreate(db, &(*newPass)[i])
            tmp := append(*oldPass, (*newPass)[i])
            oldPass = &tmp
        case model.ActionUpdate:
            status := false
            for j := range *oldPass {
                if (*newPass)[i].ID == (*oldPass)[j].ID {
                    partialUpdatePassenger(db, (*oldPass)[j].ID, &(*newPass)[i])
                    (*oldPass)[j] = (*newPass)[i]
                    status = true
                }
            }

            if !status {
                db.AddError(ErrObjectNotFound)
            }
        }
    }
}

// partialUpdateFlight updates the newFlight with all set data from newFlight. 0 or "" values means that the field should be set to nil
func partialUpdateFlight(db *gorm.DB, id uint, newFlight *model.Flight) {
    if db == nil {
        db = dh.Db
    }
    
    oldFlight := model.Flight{}
    err := dh.Db.First(&oldFlight, id).Error
    if err != nil {
        db.AddError(err)
        return
    }

    if newFlight.Description != nil {
        if *newFlight.Description == "" {
            oldFlight.Description = nil
        } else {
            oldFlight.Description = newFlight.Description
        }
    } 

    if newFlight.FuelAtDeparture != nil {
        if *newFlight.FuelAtDeparture == 0 {
            oldFlight.FuelAtDeparture = nil
        } else {
            oldFlight.FuelAtDeparture = newFlight.FuelAtDeparture
        }
    }

    db.Updates(oldFlight)
    newFlight = &oldFlight
}

func sendRealtimeEventsForPassengers(passengers []model.Passenger, defaultActionType realtime.ActionType) {
    for _, p := range passengers {
        switch p.Action {
        case model.ActionCreate:
            realtime.PassengerStream.PublishEvent(realtime.CREATED, p)
        case model.ActionUpdate:
            realtime.PassengerStream.PublishEvent(realtime.UPDATED, p)
        case model.ActionDelete:
            realtime.PassengerStream.PublishEvent(realtime.DELETED, p)
        default:
            realtime.PassengerStream.PublishEvent(defaultActionType, p)
        }
    }
}