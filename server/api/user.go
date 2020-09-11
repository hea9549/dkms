/*
 * Copyright 2019 hea9549
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"errors"
	"net/http"

	"dkms/node"
	"dkms/server/interfaces"
	"dkms/server/types"
	"dkms/share"
	"dkms/user"

	"github.com/gin-gonic/gin"
)

type User struct {
	repository  user.Repository
	nodeService node.Service
}

func NewUser(repository user.Repository, shareService node.Service) *User {
	return &User{
		repository:  repository,
		nodeService: shareService,
	}
}

func (u *User) RegisterUser(c *gin.Context) {
	var requestBody interfaces.KeyRegisterRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		BadRequestError(c, errors.New("failed to bind key register request body"))
		return
	}

	commit, err := requestBody.CommitData.ToDomain(u.nodeService.Suite)
	if err != nil {
		InternalServerError(c, err)
		return
	}

	points, err := types.EncryptedMessageToPoints(requestBody.EncryptedData, u.nodeService.GetMyPrivateKey(), u.nodeService.Suite)
	if err != nil {
		InternalServerError(c, err)
		return
	}

	yPoly, err := share.LagrangeForYPoly(u.nodeService.Suite, points[:requestBody.U], requestBody.U)
	if err != nil {
		InternalServerError(c, err)
		return
	}

	xPoly, err := share.LagrangeForXPoly(u.nodeService.Suite, points[requestBody.U:], requestBody.T)
	if err != nil {
		InternalServerError(c, err)
		return
	}

	nodes := make([]node.Node, 0)
	for _, oneNode := range requestBody.Nodes {
		n, err := oneNode.ToDomain(u.nodeService.Suite)
		if err != nil {
			InternalServerError(c, err)
			return
		}

		nodes = append(nodes, *n)
	}

	registerUser := user.User{
		Id:         requestBody.UserId,
		PolyCommit: *commit,
		MyYPoly:    *yPoly,
		MyXPoly:    *xPoly,
		Nodes:      nodes,
	}

	err = u.repository.Save(&registerUser)
	if err != nil {
		InternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, interfaces.KeyRegisterResponse{
		UserId: registerUser.Id,
		T:      xPoly.T(),
		U:      yPoly.U(),
		Commit: requestBody.CommitData,
		Nodes:  requestBody.Nodes,
	})
}
