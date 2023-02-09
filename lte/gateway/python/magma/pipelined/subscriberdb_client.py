"""
Copyright 2020 The Magma Authors.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import ipaddress
import logging
from typing import Optional

import grpc
from magma.common.service_registry import ServiceRegistry
from magma.mobilityd.utils import log_error_and_raise
from lte.protos.apn_pb2 import AggregatedMaximumBitrate
from lte.protos.subscriberdb_pb2_grpc import SubscriberDBStub
from magma.subscriberdb.sid import SIDUtils

DIRECTORYD_SERVICE_NAME = "subscriberdb"
DEFAULT_GRPC_TIMEOUT = 10
IPV4_ADDR_KEY = "ipv4_addr"

class SubscriberDbClient:

    def get_subscriber_ue_ambr(self, imsi: str) -> Optional[AggregatedMaximumBitrate]:
        """
        Make RPC call to 'GetSubscriberData' method of local SubscriberDB
        service to get the UE_AMBR.
        """

        try:
            chan = ServiceRegistry.get_rpc_channel(
                DIRECTORYD_SERVICE_NAME,
                ServiceRegistry.LOCAL,
            )
        except ValueError:
            logging.error('Cant get RPC channel to %s', DIRECTORYD_SERVICE_NAME)
            return

        client = SubscriberDBStub(chan)
        if not imsi.startswith("IMSI"):
            imsi = "IMSI" + imsi
        try:
            rsp = client.GetSubscriberData(SIDUtils.to_pb(imsi))
            return rsp.non_3gpp.ambr
    
        except ValueError as ex:
            logging.warning(
                "get_ue_ambr: Invalid or missing imsi %s: ", imsi,
            )
            logging.debug(ex)
            raise SubscriberDBImsiValueError(imsi)

        except grpc.RpcError as err:
            log_error_and_raise(
                SubscriberDBConnectionError,
                "GetSubscriberData: while reading ue_ambr error[%s] %s",
                err.code(),
                err.details(),
            )
        return None



class SubscriberDBConnectionError(Exception):
    """ Exception thrown subscriber DB is not available
    """
    pass


class SubscriberDBImsiValueError(Exception):
    """ Exception thrown when the IMSI is missing in subscriber DB.
    """
    pass

