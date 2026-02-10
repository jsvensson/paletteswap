# Changelog

## 1.0.0 (2026-02-10)


### Features

* add --version flag to pstheme-lsp ([#37](https://github.com/jsvensson/paletteswap/issues/37)) ([1720458](https://github.com/jsvensson/paletteswap/commit/1720458c96cabe71108e00c68f8fcad30bec5eea))
* add --version flag with build-time version injection ([#31](https://github.com/jsvensson/paletteswap/issues/31)) ([80a83ce](https://github.com/jsvensson/paletteswap/commit/80a83ceb074f1811ffb2157e6b5cfdd434f76312))
* add brighten() function to HCL theme format ([#11](https://github.com/jsvensson/paletteswap/issues/11)) ([38d67d4](https://github.com/jsvensson/paletteswap/commit/38d67d4e14ddae64160fc1c95125040aabff9580))
* add color.Node type with Lookup method ([136e01d](https://github.com/jsvensson/paletteswap/commit/136e01df910c846759413c7c81b918b36ea8a7b9))
* add completion, hover, go-to-definition, and document colors ([b797979](https://github.com/jsvensson/paletteswap/commit/b797979d4ccd7ef5cc9777c42fb73cc8ae570146))
* add darken() HCL function ([#15](https://github.com/jsvensson/paletteswap/issues/15)) ([2666687](https://github.com/jsvensson/paletteswap/commit/266668709d74169cc597567c7787492bf34cea48)), closes [#14](https://github.com/jsvensson/paletteswap/issues/14)
* add LSP analyzer with diagnostics and symbol table ([f87b81f](https://github.com/jsvensson/paletteswap/commit/f87b81f208c3d29a824267cbfe8cef16c5ebc8a7))
* add nested palette support with dot-notation access ([#7](https://github.com/jsvensson/paletteswap/issues/7)) ([783e602](https://github.com/jsvensson/paletteswap/commit/783e60257ad06c00f7155b4209516239fc995f1b))
* add nodeToCty conversion for palette Node ([ce2a8b9](https://github.com/jsvensson/paletteswap/commit/ce2a8b950cb17648fad9ca35f193ceb2fca18c6c))
* add pstheme-lsp skeleton with document sync ([b401326](https://github.com/jsvensson/paletteswap/commit/b401326d39b3ed7ecdc2cb50c40715b0ef3f42f8))
* add required ANSI validation and universal path template API ([#13](https://github.com/jsvensson/paletteswap/issues/13)) ([c2db3f8](https://github.com/jsvensson/paletteswap/commit/c2db3f87221357287dcccdd330985a7f303a3e50))
* add resolveColor cty helper for nested palette values ([234294d](https://github.com/jsvensson/paletteswap/commit/234294d55136fcbf51b070491ebaae0816c185c8))
* add template meta function ([03a025c](https://github.com/jsvensson/paletteswap/commit/03a025c937f8cdb6f045327ede8f4c73dec2827c))
* **lsp:** add document formatting support ([#28](https://github.com/jsvensson/paletteswap/issues/28)) ([89899dd](https://github.com/jsvensson/paletteswap/commit/89899dda3946f9ed24fb07969c725bea21ba92e3))
* **lsp:** add semantic token encoding with delta format ([fafb934](https://github.com/jsvensson/paletteswap/commit/fafb934e008ea83f980aa40c78496afe2c253050))
* **lsp:** add semantic token types and legend ([fafb934](https://github.com/jsvensson/paletteswap/commit/fafb934e008ea83f980aa40c78496afe2c253050))
* **lsp:** implement HCL AST token extraction ([fafb934](https://github.com/jsvensson/paletteswap/commit/fafb934e008ea83f980aa40c78496afe2c253050))
* **lsp:** implement semantic tokens for syntax highlighting ([#26](https://github.com/jsvensson/paletteswap/issues/26)) ([fafb934](https://github.com/jsvensson/paletteswap/commit/fafb934e008ea83f980aa40c78496afe2c253050))
* **lsp:** semantic tokenization for palette references ([#30](https://github.com/jsvensson/paletteswap/issues/30)) ([2964291](https://github.com/jsvensson/paletteswap/commit/296429165400e29390ab3f637caafd0ec37768b8))
* **lsp:** universal block reference system ([#29](https://github.com/jsvensson/paletteswap/issues/29)) ([0cd6ca8](https://github.com/jsvensson/paletteswap/commit/0cd6ca8b26ac5dd2f63788d27ff4424b50bc8ebb))
* **lsp:** wire up semantic tokens handler ([fafb934](https://github.com/jsvensson/paletteswap/commit/fafb934e008ea83f980aa40c78496afe2c253050))
* redesign template API with direct color formatting functions ([#12](https://github.com/jsvensson/paletteswap/issues/12)) ([0a8a964](https://github.com/jsvensson/paletteswap/commit/0a8a964b471916720612c8fc0cdd5229cffdae18))
* rewrite parsePaletteBody to produce *color.Node ([9798612](https://github.com/jsvensson/paletteswap/commit/9798612f9e9ab6fdd19d00979fb12d8e2092e6cc))
* wire diagnostics publishing to document sync ([3096090](https://github.com/jsvensson/paletteswap/commit/30960906d1095b8e8aa1673daaa824e86b358bcf))


### Bug Fixes

* fix ghostty template meta section ([08819a5](https://github.com/jsvensson/paletteswap/commit/08819a5fda6d7f7c702516ad20b02de8934d34d6))
* ghostty uses hex, not bhex ([b8515a4](https://github.com/jsvensson/paletteswap/commit/b8515a4896591a9ebcc7b93bbf00fd76430c4878))
* **lsp:** palette completion with incomplete references ([#27](https://github.com/jsvensson/paletteswap/issues/27)) ([84bff97](https://github.com/jsvensson/paletteswap/commit/84bff97fa5ee0b4567fd7d5f63624733734eb061))
* **lsp:** prevent stale diagnostics and nil pointer errors in VS Code ([#25](https://github.com/jsvensson/paletteswap/issues/25)) ([a02faad](https://github.com/jsvensson/paletteswap/commit/a02faadc74ea2fd325049ba2ba07b8d5c2737e57)), closes [#24](https://github.com/jsvensson/paletteswap/issues/24)
