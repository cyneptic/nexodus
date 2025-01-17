package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/nexodus-io/nexodus/internal/models"
	"github.com/stretchr/testify/assert"
)

func (suite *HandlerTestSuite) TestCreateGetDevice() {
	require := suite.Require()
	assert := suite.Assert()
	newDevice := models.AddDevice{
		OrganizationID: suite.testOrganizationID,
		PublicKey:      "atestpubkey",
	}

	resBody, err := json.Marshal(newDevice)
	require.NoError(err)

	_, res, err := suite.ServeRequest(
		http.MethodPost,
		"/", "/",
		suite.api.CreateDevice, bytes.NewBuffer(resBody),
	)
	require.NoError(err)

	body, err := io.ReadAll(res.Body)
	require.NoError(err)

	require.Equal(http.StatusCreated, res.Code, "HTTP error: %s", string(body))

	var actual models.Device
	err = json.Unmarshal(body, &actual)
	require.NoError(err)

	require.Equal(newDevice.PublicKey, actual.PublicKey)
	require.Equal(TestUserID, actual.UserID)

	_, res, err = suite.ServeRequest(
		http.MethodGet, "/:id", fmt.Sprintf("/%s", actual.ID),
		suite.api.GetDevice, nil,
	)

	require.NoError(err)
	body, err = io.ReadAll(res.Body)
	require.NoError(err)

	require.Equal(http.StatusOK, res.Code, "HTTP error: %s", string(body))

	var device models.Device
	err = json.Unmarshal(body, &device)
	require.NoError(err)

	assert.Equal(actual, device)
}

func TestChildPrefixEquals(t *testing.T) {
	tests := []struct {
		name         string
		childPrefixA []string
		childPrefixB []string
		expected     bool
	}{
		{
			name:         "identical slices",
			childPrefixA: []string{"192.168.1.0/24", "2001:db8::/32"},
			childPrefixB: []string{"192.168.1.0/24", "2001:db8::/32"},
			expected:     true,
		},
		{
			name:         "different length slices",
			childPrefixA: []string{"192.168.1.0/24", "2001:db8::/32"},
			childPrefixB: []string{"192.168.1.0/24"},
			expected:     false,
		},
		{
			name:         "different CIDR ranges",
			childPrefixA: []string{"192.168.1.0/24", "2001:db8::/32"},
			childPrefixB: []string{"192.168.2.0/24", "2001:db8::/32"},
			expected:     false,
		},
		{
			name:         "same CIDR ranges, different order",
			childPrefixA: []string{"192.168.1.0/24", "2001:db8::/32"},
			childPrefixB: []string{"2001:db8::/32", "192.168.1.0/24"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := childPrefixEquals(tt.childPrefixA, tt.childPrefixB)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
