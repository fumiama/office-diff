# office-diff

Diff tool for OpenXML Office files

## About The Project

**office-diff** is a diff tool for OpenXML Office files that shows differences on code more precisely xml level.

While working on various Office file editing projects, I found it very difficult to figure out why some processes
change or even damage a file.  
First I extracted the files, formatted the whole xml and used git to compare the two extracted directories. But I
thought it might be helpful to have a tool that does all this automatically (without the git dependency).  
For this reason, I have developed _office-diff_. This tool extracts the specified files to a temporary location,
formats the xml and outputs a unified diff.

## Usage

```shell
# output to stdout
office-diff <file1> <file2>

# output to file
office-diff <file1> <file2> --output result.diff
```

## Roadmap

See the [open issues](https://github.com/develerik/office-diff/issues) for a list of proposed features
(and known issues).

## Get Support

This project is maintained by [@develerik](https://github.com/develerik). Please understand that I won't be able to
provide individual support via email. I also believe that help is much more valuable if it's shared publicly, so that
more people can benefit from it.

- [**Report a bug**](https://github.com/develerik/office-diff/issues/new)
- [**Requests a new
feature**](https://github.com/develerik/office-diff/issues/new)
- [**Report a security
vulnerability**](https://github.com/develerik/office-diff/issues/new)

## Contributing

Contributions are what make the open source community such an amazing place to be learn, inspire, and create. Any
contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct.

## Maintainers

- **Erik Bender** - *Initial work* - [develerik](https://github.com/develerik)

See also the list of [contributors](https://github.com/develerik/office-diff/graphs/contributors) who participated in
this project.

## License

Distributed under the ISC License. See [LICENSE](LICENSE) for more information.
