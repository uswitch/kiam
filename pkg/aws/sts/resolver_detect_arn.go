// Portions of below were based on kube2iam (https://github.com/jtblin/kube2iam). It's
// license is copied below:
// Copyright (c) Jerome Touffe-Blin ("Author")
// All rights reserved.

// The BSD License

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:

// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.

// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.

// THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
// PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR
// BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE
// OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN
// IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package sts

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

// InstanceProfileArn uses the EC2 metadata API to find the role for
// the instance.
func InstanceProfileArn() (string, error) {
	sess := session.Must(session.NewSession())
	svc := ec2metadata.New(sess)
	if !svc.Available() {
		return "", fmt.Errorf("aws metadata api not available")
	}

	info, err := svc.IAMInfo()
	if err != nil {
		return "", fmt.Errorf("error accessing iam info: %s", err)
	}

	return info.InstanceProfileArn, nil
}

// RoleName parses out the role name from the instance profile arn
func RoleName(instanceProfileArn string) (string, error) {
	parts := strings.Split(instanceProfileArn, "/")

	roleName := strings.Join(parts[1:], "/")
	return roleName, nil
}

// DetectRoleName uses the EC2 metadata API to detect the name of the 
// role from the instance profile
func DetectRoleName() (string, error) {
	instanceArn, err := InstanceProfileArn()
	if err != nil {
		return "", err
	}

	return RoleName(instanceArn)
}

// BaseArn calculates the base arn given an instance's arn
func BaseArn(instanceProfileArn string) (string, error) {
	// instance profile arn will be of the form:
	// arn:aws:iam::account-id:instance-profile/role-name
	// so we use the instance-profile prefix as the prefix for our roles

	parts := strings.Split(instanceProfileArn, ":")
	accountPrefix := strings.Join(parts[0:5], ":")

	return fmt.Sprintf("%s:role/", accountPrefix), nil
}

// DetectARNPrefix uses the EC2 metadata API to determine the
// current prefix.
func DetectARNPrefix() (string, error) {
	instanceArn, err := InstanceProfileArn()
	if err != nil {
		return "", err
	}

	return BaseArn(instanceArn)
}
