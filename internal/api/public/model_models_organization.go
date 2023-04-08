/*
Nexodus API

This is the Nexodus API Server.

API version: 1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package public

// ModelsOrganization struct for ModelsOrganization
type ModelsOrganization struct {
	Cidr        string             `json:"cidr,omitempty"`
	CidrV6      string             `json:"cidr_v6,omitempty"`
	Description string             `json:"description,omitempty"`
	HubZone     bool               `json:"hub_zone,omitempty"`
	Id          string             `json:"id,omitempty"`
	Invitations []ModelsInvitation `json:"invitations,omitempty"`
	Name        string             `json:"name,omitempty"`
	OwnerId     string             `json:"owner_id,omitempty"`
}