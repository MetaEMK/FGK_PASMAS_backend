package passenger

import (
	"net/http"

	"github.com/MetaEMK/FGK_PASMAS_backend/logging"
	"github.com/MetaEMK/FGK_PASMAS_backend/router/api"
	"github.com/MetaEMK/FGK_PASMAS_backend/service/passengerService"
	"github.com/gin-gonic/gin"
)

var log = logging.ApiLogger

func getPassengers(c *gin.Context) {
    passengers, err := passengerService.GetPassengers()

    if err != nil {
        apiErr := api.GetErrorResponse(err)
        apiErr.ErrorResponse.Message = err.Error()
        c.JSON(apiErr.HttpCode, apiErr.ErrorResponse)
    } else {
        c.JSON(http.StatusOK, api.SuccessResponse { Success: true, Response: passengers })
    }

}

