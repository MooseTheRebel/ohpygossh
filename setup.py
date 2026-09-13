import json
import os
import subprocess
import sys
from distutils.core import Extension
from pathlib import Path

import setuptools
from setuptools.command.build_ext import build_ext

"""
NOTE: This project's 'setup.py' branches on the CIBUILDWHEEL env var,
(set to "1" by cibuildwheel). This is documented in cibuildwheel's faq
page: https://cibuildwheel.pypa.io/en/v4.2.1/faq/#:~:text=the%20environment%20variable%20CIBUILDWHEEL

Additionally:

* Under cibuildwheel, each platform build is isolated, so 'gopy build' must be
  invoked from here via a custom build_ext step.
* For local/manual builds, 'gopy build' is run separately beforehand (see
  make_and_validate_script.sh), so this script just packages the already-built output.

"""

PACKAGE_PATH = "gohpygossh"
PACKAGE_NAME = "ohpygossh"

this_directory = Path(__file__).parent
long_description = (this_directory / "README.md").read_text()
version = (this_directory / "version.txt").read_text(encoding="utf-8").strip()

common_kwargs = {
    "name": PACKAGE_NAME,
    "version": version,
    "author": "b-long",
    "author_email": "b-long@users.noreply.github.com",
    "description": "A project to create a new cross-platform SSH wheel for Python.",
    "long_description": long_description,
    "long_description_content_type": "text/markdown",
    "url": "https://github.com/b-long/ohpygossh",
    "license": "MIT",
    "classifiers": [
        "Programming Language :: Python :: 3",
        "Operating System :: OS Independent",
    ],
    "include_package_data": True,
}

if os.getenv("CIBUILDWHEEL") == "1":
    if sys.platform == "darwin":
        # PYTHON_BINARY_PATH is set explicitly for 310 and 311, see build_wheel.yml
        # on macos PYTHON_BINARY_PATH must be python bin installed from python.org or from brew
        PYTHON_BINARY = os.getenv("PYTHON_BINARY_PATH", sys.executable)
        if PYTHON_BINARY == sys.executable:
            subprocess.run(
                [sys.executable, "-m", "pip", "install", "pybindgen"], check=False
            )
    else:
        # linux & windows
        # cibuildwheel v3 invokes setup.py in a minimal isolated venv without pip to
        # discover build requirements; run() instead of check_call() so the process
        # doesn't abort when pip is unavailable. pybindgen is pre-installed via
        # CIBW_BEFORE_BUILD in the actual build environment.
        PYTHON_BINARY = sys.executable
        subprocess.run(
            [sys.executable, "-m", "pip", "install", "pybindgen"], check=False
        )

    def _generate_path_with_gopath() -> str:
        go_path = (
            subprocess.check_output(["go", "env", "GOPATH"]).decode("utf-8").strip()
        )
        path_val = f"{os.getenv('PATH')}:{go_path}/bin"
        return path_val

    class CustomBuildExt(build_ext):
        def build_extension(self, ext: Extension):
            bin_path = _generate_path_with_gopath()
            go_env = json.loads(
                subprocess.check_output(["go", "env", "-json"]).decode("utf-8").strip()
            )

            destination = (
                os.path.dirname(os.path.abspath(self.get_ext_fullpath(ext.name)))
                + f"/{PACKAGE_NAME}"
            )

            subprocess.check_call(
                [
                    "gopy",
                    "build",
                    "-no-make",
                    "-dynamic-link=True",
                    "-output",
                    destination,
                    "-vm",
                    PYTHON_BINARY,
                    *ext.sources,
                ],
                env={"PATH": bin_path, **go_env, "CGO_LDFLAGS_ALLOW": ".*"},
            )

            # dirty hack to avoid "from pkg import pkg", remove if needed
            with open(f"{destination}/__init__.py", "w") as f:
                f.write(f"from .{PACKAGE_PATH} import *")

    setuptools.setup(
        **common_kwargs,
        cmdclass={
            "build_ext": CustomBuildExt,
        },
        ext_modules=[
            Extension(
                PACKAGE_NAME,
                [PACKAGE_PATH],
            )
        ],
    )
else:
    setuptools.setup(
        **common_kwargs,
        packages=setuptools.find_packages(include=[PACKAGE_NAME]),
        py_modules=[f"{PACKAGE_NAME}.{PACKAGE_PATH}"],
        package_data={PACKAGE_NAME: ["*.so"]},
    )
