/*
Copyright 2022 The Magma Authors.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grant

import (
	"magma/dp/cloud/go/services/dp/active_mode_controller/protos/active_mode"
)

func GetFrequencyGrantMapping(grants []*active_mode.Grant) map[int64]*active_mode.Grant {
	m := make(map[int64]*active_mode.Grant, len(grants))
	for _, g := range grants {
		m[(g.HighFrequencyHz+g.LowFrequencyHz)/2] = g
	}
	return m
}