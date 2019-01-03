# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from subprocess import STDOUT, CalledProcessError
from typing import Callable, Tuple

from telepresence.cli import Args, PortMapping
from telepresence.proxy import RemoteInfo
from telepresence.runner import Runner, launch_local_server
from telepresence.utilities import find_free_port

from .expose import expose_local_services
from .ssh import SSH


def connect(
    runner: Runner, remote_info: RemoteInfo, is_container_mode: bool,
    expose: PortMapping
) -> Tuple[int, SSH]:
    """
    Start all the processes that handle remote proxying.

    Return (local port of SOCKS proxying tunnel, SSH instance).
    """
    span = runner.span()
    # Keep local copy of pod logs, for debugging purposes:
    runner.launch(
        "kubectl logs",
        runner.kubectl(
            "logs", "-f", remote_info.pod_name, "--container",
            remote_info.container_name, "--tail=10"
        ),
        bufsize=0,
    )

    ssh = SSH(runner, find_free_port())

    # forward remote port to here, by tunneling via remote SSH server:
    runner.launch(
        "kubectl port-forward",
        runner.kubectl(
            "port-forward", remote_info.pod_name, "{}:8022".format(ssh.port)
        )
    )

    ssh.wait()

    # In Docker mode this happens inside the local Docker container:
    if not is_container_mode:
        expose_local_services(
            runner,
            ssh,
            list(expose.local_to_remote()),
        )

    # Start tunnels for the SOCKS proxy (local -> remote)
    # and the local server for the proxy to poll (remote -> local).
    socks_port = find_free_port()
    local_server_port = find_free_port()

    launch_local_server(runner, local_server_port)
    forward_args = [
        "-L127.0.0.1:{}:127.0.0.1:9050".format(socks_port),
        "-R9055:127.0.0.1:{}".format(local_server_port)
    ]
    runner.launch(
        "SSH port forward (socks and proxy poll)",
        ssh.bg_command(forward_args)
    )

    span.end()
    return socks_port, ssh


def setup(runner: Runner, args: Args) -> Callable[[Runner, RemoteInfo], Tuple[int, SSH]]:
    # Make sure we can run openssh:
    runner.require(["ssh"], "Please install the OpenSSH client")
    try:
        version = runner.get_output(["ssh", "-V"], stderr=STDOUT)
        if not version.startswith("OpenSSH"):
            raise runner.fail("'ssh' is not the OpenSSH client, apparently.")
    except (CalledProcessError, OSError, IOError) as e:
        raise runner.fail("Error running ssh: {}\n".format(e))

    is_container_mode = args.method == "container"

    def do_connect(runner_: Runner, remote_info: RemoteInfo) -> Tuple[int, SSH]:
        return connect(runner_, remote_info, is_container_mode, args.expose)

    return do_connect
