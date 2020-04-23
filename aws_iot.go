package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/aws/aws-sdk-go/service/iot/iotiface"
)

var (
	name      string
	thingType string
	svc       iotiface.IoTAPI
)

func newAWSConfig() *aws.Config {
	c := aws.NewConfig().WithRegion("ap-south-1")
	return c
}

// NewClient cerates a new AWS IoT client
func NewClient() {
	svc = iot.New(session.New(), &aws.Config{
		Region:      aws.String("ap-south-1"),
		Credentials: credentials.NewStaticCredentials("<access id>", "<secret key>", ""),
	})
}

// DescribeThing gets the info about existing THING
func DescribeThing(name string) {
	resp, err := svc.DescribeThing(&iot.DescribeThingInput{ThingName: aws.String(name)})

	if err != nil {
		fmt.Printf("Failed to describe thing: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Thing Ver: ", *resp.Version)
	fmt.Println("Thing Name: ", *resp.ThingName)
}

// DeleteThing deletes the given THING
func DeleteThing(ver int64, name string) {
	resp, err := svc.DeleteThing(&iot.DeleteThingInput{ExpectedVersion: aws.Int64(ver), ThingName: aws.String(name)})

	if err != nil {
		fmt.Printf("Failed to delete %s: %v\n\n", name, err)
		os.Exit(1)
	}

	fmt.Println("Thing Ver: ", resp.GoString())
}

// RegisterCACertificate registers a CA certificate with AWS IoT.
// This CA certificate can then be used to sign device certificates.
func RegisterCACertificate(rootCert string, veriCert string) {
	// read the CA certificate
	cert, err := ioutil.ReadFile(rootCert)
	if err != nil {
		fmt.Println("Failed to read CA file ", err)
		return
	}

	//read the verification certificate
	verCert, err := ioutil.ReadFile(veriCert)
	if err != nil {
		fmt.Println("Failed to read verification file ", err)
		return
	}
	resp, err := svc.RegisterCACertificate(setRegisterCACertificateInput(string(cert), string(verCert)))

	if err != nil {
		fmt.Printf("Failed to register CA cert: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Cert ARN: ", *resp.CertificateArn)
	fmt.Println("Cert ID: ", *resp.CertificateId)
}

// DeRegisterCACert deregisters the CA cert
func DeRegisterCACert(certid string) {

	var certInput iot.UpdateCACertificateInput
	certInput.SetCertificateId(certid)
	certInput.SetNewAutoRegistrationStatus("INACTIVE")
	certInput.SetNewStatus(iot.CACertificateStatusInactive)
	certInput.SetRemoveAutoRegistration(true)

	_, err := svc.UpdateCACertificate(&certInput)

	if err != nil {
		fmt.Printf("Failed to update CA cert: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("CA cert deactivated\n")
}

// RegisterThing registers a new dev cert with AWS IoT
func RegisterThing(name string, caCert string, devCert string) {

	cacert, err := ioutil.ReadFile(caCert)
	if err != nil {
		fmt.Println("Failed to read CA cert ", err)
		return
	}

	devcert, err := ioutil.ReadFile(devCert)
	if err != nil {
		fmt.Println("Failed to read dev cert ", err)
		return
	}

	provisionParam := map[string]*string{
		"ThingName":        aws.String(name),
		"SerialNumber":     aws.String("0x12345"),
		"CACertificatePem": aws.String(string(cacert)),
		"CertificatePem":   aws.String(string(devcert)),
	}

	templateBody, err := ioutil.ReadFile("template.json")
	if err != nil {
		fmt.Printf("Failed to read file template.json")
		os.Exit(1)
	}

	templateString := string(templateBody[:])

	resp, err := svc.RegisterThing(
		&iot.RegisterThingInput{
			Parameters:   provisionParam,
			TemplateBody: &templateString,
		})

	if err != nil {
		fmt.Printf("Failed to create thing: %v", err)
		os.Exit(1)
	}

	fmt.Println("Thing ARN: ", resp.GoString())
}

// DeRegisterDevCert deregisters the device cert
func DeRegisterDevCert(devcertid string) {

	var certInput iot.UpdateCertificateInput
	certInput.SetCertificateId(devcertid)
	certInput.SetNewStatus(iot.CACertificateStatusInactive)

	_, err := svc.UpdateCertificate(&certInput)

	if err != nil {
		fmt.Printf("Failed to deregister device cert: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Device cert deactivated\n")
}

// setAttributePayload is used to set parameters for thing creation
func setAttributePayload() *iot.AttributePayload {

	attr := &iot.AttributePayload{
		Attributes: map[string]*string{},
	}

	// As of now, we use the same thing type for all devices
	thingType = "testthing"
	attr.Attributes["Type"] = aws.String(thingType)

	return attr
}

// setRegisterCACertificateInput is used to set parameters for CA cert registration
func setRegisterCACertificateInput(cert string, verCert string) *iot.RegisterCACertificateInput {

	certInput := new(iot.RegisterCACertificateInput)
	certInput.SetAllowAutoRegistration(true)
	certInput.SetCaCertificate(cert)
	certInput.SetSetAsActive(true)
	certInput.SetVerificationCertificate(verCert)

	err := certInput.Validate()
	if err != nil {
		fmt.Println("Error in registering CA cert: ", err)
	}

	return certInput
}

func main() {

	// connect to AWS platform
	NewClient()

	// Register the CA cert
	RegisterCACertificate("rootCA.pem", "verificationCert.pem")

	// Register the device with name iot-dev-1
	RegisterThing("iot-dev-1", "rootCA.pem", "devCert.pem")

	// Retrive the details of device named iot-dev-1
	DescribeThing("iot-dev-1")
}
