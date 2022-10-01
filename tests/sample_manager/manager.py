import typing
from .sample import Sample
from .sample1 import s1


class SampleManager():
    def __init__(self, samples: typing.Dict[str, Sample]) -> None:
        self._samples = samples

    @property
    def simple_project(self) -> Sample:
        return self._samples["simple_project"]


samples: typing.Dict[str, Sample] = {}
samples["simple_project"] = s1

sm = SampleManager(samples)