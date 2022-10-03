import logging
import subprocess
from sample_manager import sm


def test_basic_go_mod_build(tmp_path):
    logging.info("build simple go.mod project")
    sm.simple_go_mod_sample.generate(tmp_path)
    res = subprocess.run(["gococo", "build"], capture_output=True, cwd=tmp_path)
    assert res.returncode == 0
    assert res.stdout.find(b'information parsed') > 0

def test_basic_go_work_build(tmp_path):
    logging.info("build simple go.work project")
    sm.simple_go_work_sample.generate(tmp_path)
    res = subprocess.run(["gococo", "build", "-o", "haha", "example.com/hello"], capture_output=True, cwd=tmp_path)
    assert res.returncode == 0
    assert res.stdout.find(b'information parsed') > 0

    logging.info("check the built binary")

    logging.info("check the second compile")
    res = subprocess.run(["gococo", "build", "-o", "haha", "example.com/hello"], capture_output=True, cwd=tmp_path)
    assert res.returncode == 0
    assert res.stdout.find(b'no need to copy, using cached project') > 0
