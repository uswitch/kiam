// Copyright 2017 uSwitch
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sts

import (
	"time"
)

type Credentials struct {
	Code            string
	Type            string
	AccessKeyId     string
	SecretAccessKey string
	Token           string
	Expiration      string
	LastUpdated     string
}

const timeLayout = "2006-01-02T15:04:05Z"

func NewCredentials(accessKey, secretKey, token string, expiry time.Time) *Credentials {
	return &Credentials{
		Code:            "Success",
		Type:            "AWS-HMAC",
		LastUpdated:     time.Now().Format(timeLayout),
		AccessKeyId:     accessKey,
		SecretAccessKey: secretKey,
		Token:           token,
		Expiration:      expiry.Format(timeLayout),
	}
}
