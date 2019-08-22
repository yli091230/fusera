package gps

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Locator Interface that describes everything needed to describe a location for the SDL API.
type Locator interface {
	SdlCloudName() string
	Region() (string, error)
	Locality() string
	LocalityType() string
}

// GcpLocation A location for GCP environment.
type GcpLocation struct{}

// SdlCloudName Returns gs, the proper string SDL associates with GCP.
func (g *GcpLocation) SdlCloudName() string {
	return "gs"
}

// Region Returns the sublocation of the cloud platform the current server is running on.
func (g *GcpLocation) Region() (string, error) {
	region, err := resolveGcpZone()
	if err != nil {
		return "", err
	}
	return region, nil
}

// Locality Returns the locality for GCP environment.
func (g *GcpLocation) Locality() string {
	token, err := retrieveGCPInstanceToken()
	if err != nil {
		return ""
	}
	return string(token)
}

// LocalityType Returns the locality-type for GCP environment.
func (g *GcpLocation) LocalityType() string {
	return "gcp_jwt"
}

// AwsLocation A location for AWS environment.
type AwsLocation struct{}

// SdlCloudName Returns s3, the proper string SDL associates with AWS.
func (a *AwsLocation) SdlCloudName() string {
	return "s3"
}

// Region Returns the sublocation of the cloud platform the current server is running on.
func (a *AwsLocation) Region() (string, error) {
	region, err := resolveAwsRegion()
	if err != nil {
		return "", err
	}
	return region, nil
}

// Locality Returns the locality for AWS environment. //TODO: Implement
func (a *AwsLocation) Locality() string {
	return ""
}

// LocalityType Returns the locality-type for AWS environment.
func (a *AwsLocation) LocalityType() string {
	return "aws_pkcs7"
}

// TODO: try to be more siphisticated in figuring out if location is ncbi or follows cloud.region format

// ManualLocation A location for a manual environment.
type ManualLocation struct {
	locality string
}

// SdlCloudName Returns whatever it was given as the cloud name.
func (m *ManualLocation) SdlCloudName() string {
	return m.locality
}

// Region Returns the sublocation of the cloud platform the current server is running on.
func (m *ManualLocation) Region() (string, error) {
	return m.locality, nil
}

// Locality Returns the locality for a manual environment.
func (m *ManualLocation) Locality() string {
	return m.locality
}

// LocalityType Returns the locality-type "forced" for a manual environment.
func (m *ManualLocation) LocalityType() string {
	return "forced"
}

// NewManualLocation Returns a new manual location with the provided platform.
func NewManualLocation(location string) (*ManualLocation, error) {
	return &ManualLocation{locality: location}, nil
}

// GenerateLocator Determines which locator to use by attempting to detect what cloud platform it is running on.
func GenerateLocator() (Locator, error) {
	_, err := resolveAwsRegion()
	if err != nil {
		// could be on google
		// retain aws error message
		msg := err.Error()
		_, err := retrieveGCPInstanceToken()
		if err != nil {
			// return both aws and google error messages
			return nil, errors.Wrap(err, msg)
		}
		return &GcpLocation{}, nil
	}
	return &AwsLocation{}, nil
}

// ResolveTraditionalLocation Forms the traditional location string.
// func ResolveTraditionalLocation() (string, error) {
// 	platform, err := ResolveRegion()
// 	if err != nil {
// 		return "", err
// 	}
// 	return platform.Name + "." + platform.Region, nil
// }

// FindLocation Attempts to find the location fusera is running, be it GCP or AWS.
// If location cannot be resolved, return error.
// FindLocation attempts to figure out which cloud
// provider Fusera is running on and what region of that cloud.
// func FindLocation() (*Platform, error) {
// 	p := &Platform{}
// 	aws, err := resolveAwsRegion()
// 	if err != nil {
// 		// could be on google
// 		// retain aws error message
// 		msg := err.Error()
// 		token, err := retrieveGCPInstanceToken()
// 		if err != nil {
// 			// return both aws and google error messages
// 			return nil, errors.Wrap(err, msg)
// 		}
// 		zone, err := resolveGcpZone()
// 		if err != nil {
// 			// return both aws and google error messages
// 			return nil, errors.Wrap(err, msg)
// 		}
// 		p.Name = "gs"
// 		p.Region = zone
// 		p.InstanceToken = token
// 		return p, nil
// 	}
// 	p.Name = "s3"
// 	p.Region = aws
// 	return p, nil
// }

// ResolveRegion Attempt to resolve the location on aws or gs.
// func ResolveRegion() (*Platform, error) {
// 	platform := &Platform{}
// 	region, err := resolveAwsRegion()
// 	if err != nil {
// 		// could be on google
// 		// retain aws error message
// 		msg := err.Error()
// 		region, err = resolveGcpZone()
// 		if err != nil {
// 			// return both aws and google error messages
// 			return nil, errors.Wrap(err, msg)
// 		}
// 		platform.Name = "gs"
// 		platform.Region = region
// 		return platform, nil
// 	}
// 	platform.Name = "s3"
// 	platform.Region = region
// 	return platform, nil
// }

func resolveAwsRegion() (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 1 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       500 * time.Millisecond,
			TLSHandshakeTimeout:   500 * time.Millisecond,
			ExpectContinueTimeout: 500 * time.Millisecond,
		},
	}
	// maybe we are on an AWS instance and can resolve what region we are in.
	// let's try it out and if we timeout we'll return an error.
	// use this url: http://169.254.169.254/latest/dynamic/instance-identity/document
	resp, err := client.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")
	if err != nil {
		return "", errors.Wrapf(err, "location was not provided, fusera attempted to resolve region but encountered an error, this feature only works when fusera is on an amazon or google instance")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("issue trying to resolve region, got: %d: %s", resp.StatusCode, resp.Status)
	}
	var payload struct {
		Region string `json:"region"`
	}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return "", errors.New("issue trying to resolve region, couldn't decode response from amazon")
	}
	if payload.Region == "" {
		return "", errors.New("issue trying to resolve region, amazon returned empty region")
	}
	return payload.Region, nil
}

func resolveGcpZone() (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 1 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       500 * time.Millisecond,
			TLSHandshakeTimeout:   500 * time.Millisecond,
			ExpectContinueTimeout: 500 * time.Millisecond,
		},
	}
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone?alt=json", nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "location was not provided, fusera attempted to resolve region but encountered an error, this feature only works when fusera is on an amazon or google instance")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("issue trying to resolve region, got: %d: %s", resp.StatusCode, resp.Status)
	}
	var payload string
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return "", errors.New("issue trying to resolve region, couldn't decode response from google")
	}
	path := filepath.Base(payload)
	if path == "" || len(path) == 1 {
		return "", errors.New("issue trying to resolve region, google returned empty region")
	}
	return path, nil
}

func retrieveGCPInstanceToken() ([]byte, error) {
	// make a request to token endpoint
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 1 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       500 * time.Millisecond,
			TLSHandshakeTimeout:   500 * time.Millisecond,
			ExpectContinueTimeout: 500 * time.Millisecond,
		},
	}
	req, err := http.NewRequest("GET", "http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=https://www.ncbi.nlm.nih.gov&format=full", nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "location was not provided, fusera attempted to resolve region but encountered an error, this feature only works when fusera is on an amazon or google instance")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("issue trying to retreive GCP instance token, got: %d: %s", resp.StatusCode, resp.Status)
	}
	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("issue trying to resolve region, couldn't decode response from google")
	}
	return token, nil
}
