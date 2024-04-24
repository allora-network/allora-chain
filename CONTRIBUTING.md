Any contribution that you make to this repository will
be under the Apache 2 License, as dictated by that
[license](http://www.apache.org/licenses/LICENSE-2.0.html):

~~~
5. Submission of Contributions. Unless You explicitly state otherwise,
   any Contribution intentionally submitted for inclusion in the Work
   by You to the Licensor shall be under the terms and conditions of
   this License, without any additional terms or conditions.
   Notwithstanding the above, nothing herein shall supersede or modify
   the terms of any separate license agreement you may have executed
   with Licensor regarding such Contributions.
~~~

Contributors must sign-off each commit by adding a `Signed-off-by: ...`
line to commit messages to certify that they have the right to submit
the code they are contributing to the project according to the
[Developer Certificate of Origin (DCO)](https://developercertificate.org/).

Contributors are encouraged to enable our pre-commit hooks to facilitate
teh passage of CI/CD-related checks in our repositories. To enable them,
please run the following command in the root of the repository:

```bash
chmod +x .hooks/pre-commit
git config core.hooksPath .hooks/pre-commit 
```

This will enable the pre-commit hooks for the repository and ensure that
all commits are checked for compliance with our formatter and other
checks before they are accepted into the repository.
