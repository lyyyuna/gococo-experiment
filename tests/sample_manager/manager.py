import typing
from .sample import Sample
from .simple_go_sample import simple_go_mod_sample, simple_go_work_sample


class SampleManager():
    def __init__(self, samples: typing.Dict[str, Sample]) -> None:
        self._samples = samples

    @property
    def simple_go_mod_sample(self) -> Sample:
        return self._samples["simple_go_mod_sample"]

    @property
    def simple_go_work_sample(self) -> Sample:
        return self._samples["simple_go_work_sample"]


samples: typing.Dict[str, Sample] = {}
samples["simple_go_mod_sample"] = simple_go_mod_sample
samples["simple_go_work_sample"] = simple_go_work_sample

sm = SampleManager(samples)