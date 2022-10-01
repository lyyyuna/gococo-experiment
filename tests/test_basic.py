import logging
import subprocess
from sample_manager import sm


def test_basic_build(tmp_path):
    sm.simple_project.generate(tmp_path)
    res = subprocess.run(["gococo", "build"], capture_output=True, cwd=tmp_path)
    assert res.returncode == 0
    assert res.stdout.find(b'information parsed') > 0
