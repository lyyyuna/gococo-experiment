import os
import typing


class Sample():
    def __init__(self) -> None:
        self.files: typing.Dict[str, str] = {}

    def __setitem__(self, path: str, content: str):
        self.files[path] = content

    def __getitem__(self, path: str) -> str:
        return self.files[path]

    def generate(self, base: str):
        '''
        Generate a real project in your local directory

        base -- the base folder
        '''
        for path, content in self.files.items():
            filepath = os.path.join(base, path)
            with open(filepath, 'w+') as f:
                f.write(content)
