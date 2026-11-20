# Warning — educational use only

This project was created **strictly for educational and lawful security-research
purposes**. Do not misuse it. Only use it on hardware you own or are explicitly
authorised in writing to test. See [SECURITY.md](SECURITY.md) and the
disclaimer in [README.md](README.md).

## Upstream compatibility warning (verbatim)

The project this toolkit was compared against,
[`sirbits/wrice-zlt-s20-unlocker`](https://github.com/sirbits/wrice-zlt-s20-unlocker),
ships the following warning in `src/warning`. It is reproduced here verbatim
because it is important context for anyone testing these devices:

> zlt s20 source...
> this code was tested specifically on two models running specific apis.
> the apis have long been patched, so it might not work for your specific device.
> I am not responsible if you brick your device. do not contact me for repairs.

Compatibility depends on firmware version. Nothing here is a guarantee that a
given device will respond — always use `--dry-run=true` first and keep a backup.
