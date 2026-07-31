# testdata

Samples files come from [here](http://techslides.com/sample-files-for-development)

To write tags to files you can use `lltag`:

```sh
lltag sample.* \
  -a "Test Artist" \
  -t "Test Title" \
  -A "Test Album" \
  -n "3" \
  -g "Jazz" \
  -d "2000" \
  -c "Test Comment" \
  --tag ALBUMARTIST="Test AlbumArtist" \
  --tag COMPOSER="Test Composer"\
  --tag DISCNUMBER="02" \
  --tag TRACKTOTAL="06"
```

## Cover art fixtures

The `test{VORBIS,MP4,FLAC,MP3v23}_*Covers.*` and `test*_coverArt.*` files are the
cover-art samples from the Apache Tika project and are used to test picture
extraction (single and multiple covers).

## Ogg codec fixtures

`testFLAC.oga` (FLAC-in-Ogg) and `testOPUS.opus` are from the Apache Tika
project. `testSPEEX.spx` (Speex) and `testTHEORA.ogv` (Theora) were generated
with ffmpeg. They test tag reading across the Ogg-mapped codecs:

```sh
ffmpeg -f lavfi -i "sine=frequency=440:duration=1" -c:a libspeex \
  -metadata title="Test Title" -metadata artist="Test Artist" -metadata album="Test Album" testSPEEX.spx
ffmpeg -f lavfi -i "testsrc=duration=1:size=32x32:rate=1" -c:v libtheora \
  -metadata title="Test Title" -metadata artist="Test Artist" -metadata album="Test Album" testTHEORA.ogv
```
