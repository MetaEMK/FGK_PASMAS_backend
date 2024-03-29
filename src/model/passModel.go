package model

import (
	"gorm.io/gorm"
)

type Passenger struct {
    gorm.Model
    LastName            string
    FirstName           string
    Weight              uint            `gorm:"not null"`

    FlightID            uint            `gorm:"index"`
    Flight              *Flight         `gorm:"foreignKey:FlightID"`

    // This virtual field is used in the api to determine what action to take on this passenger
    Action              Action          `gorm:"-"`
}

type Action string

const (
    ActionCreate Action = "CREATE"
    ActionUpdate Action = "UPDATE"
    ActionDelete Action = "DELETE"
)
