package model

import (
	"time"

	"gorm.io/gorm"
)

type Flight struct {
    gorm.Model
    Status               FlightType
    Description         *string
    FuelAtDeparture     *float32
    DepartureTime       time.Time
    ArrivalTime         time.Time

    PlaneId             uint                
    Plane               *Plane                  `gorm:"foreignKey:PlaneId"`
    PilotId             *uint
    Pilot               *Pilot                  `gorm:"foreignKey:PilotId"`
    Passengers          *[]Passenger            `gorm:"foreignKey:FlightID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}


type FlightType string

const (
    FsPlanned = "PLANNED"
    FsReserved = "RESERVED"
    FsBooked = "BOOKED"
    FsBlocked = "BLOCKED"
)
