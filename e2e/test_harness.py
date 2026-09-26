# SPDX-License-Identifier: AGPL-3.0-only
"""Tests of the harness itself: stopping the server loses no output."""

import threading
import time

from conftest import Server


class FinishedProcess:
    """A process that has already exited with status 0."""

    def terminate(self) -> None:
        pass

    def wait(self, timeout=None) -> int:
        return 0


def test_stop_waits_for_the_last_line_of_output():
    output = ['{"message":"serving"}']

    def slow_pump() -> None:
        # The process has exited, but its last line is still in the pipe.
        time.sleep(0.3)
        output.append('{"message":"stopped"}')

    pump = threading.Thread(target=slow_pump)
    pump.start()
    server = Server(process=FinishedProcess(), startup={}, base_url="", ready_seconds=0.0,
                    output=output, pump=pump)
    code, later = server.stop()
    assert code == 0
    assert [line["message"] for line in later] == ["stopped"]
