# Changelog

## [0.1.4](https://github.com/jsvensson/paletteswap/compare/v0.1.3...v0.1.4) (2026-02-12)


### Features

* **cli:** add fmt subcommand for formatting .pstheme files ([#58](https://github.com/jsvensson/paletteswap/issues/58)) ([87fe7dc](https://github.com/jsvensson/paletteswap/commit/87fe7dcd4181ab8fb1cf8aca79ec92f042c8665c))
* **lsp:** collapse multiple blank lines to one when formatting ([75c7c51](https://github.com/jsvensson/paletteswap/commit/75c7c51af5f8b55d5c57248ba2e58697f5a97e84))
* **lsp:** collapse multiple blank lines when formatting ([#56](https://github.com/jsvensson/paletteswap/issues/56)) ([75c7c51](https://github.com/jsvensson/paletteswap/commit/75c7c51af5f8b55d5c57248ba2e58697f5a97e84))


### CI/CD

* add test workflow for pull requests ([#57](https://github.com/jsvensson/paletteswap/issues/57)) ([8cfcb2a](https://github.com/jsvensson/paletteswap/commit/8cfcb2aaa2d668446f7b99a8a7ac89307be1ef4c))
* add workflow_dispatch to release workflow ([#54](https://github.com/jsvensson/paletteswap/issues/54)) ([55cb776](https://github.com/jsvensson/paletteswap/commit/55cb77623521ad725cb6af503ca57429995499d5))

## [0.1.3](https://github.com/jsvensson/paletteswap/compare/v0.1.2...v0.1.3) (2026-02-12)


### CI/CD

* add missing issues:write permission for release-please ([#51](https://github.com/jsvensson/paletteswap/issues/51)) ([24826ce](https://github.com/jsvensson/paletteswap/commit/24826ceb122a03930234f1051d19ebb035969150))
* add pstheme-lsp binary to release builds ([#49](https://github.com/jsvensson/paletteswap/issues/49)) ([34c3ed5](https://github.com/jsvensson/paletteswap/commit/34c3ed5c87070fdb5205583ba72b867f87423fba))
* exclude ci commits from triggering release-please ([#52](https://github.com/jsvensson/paletteswap/issues/52)) ([3ee7cdd](https://github.com/jsvensson/paletteswap/commit/3ee7cdda093c991786b15496618633640c17de63))
* fix release workflow to handle release-please PRs ([#48](https://github.com/jsvensson/paletteswap/issues/48)) ([6cd24e7](https://github.com/jsvensson/paletteswap/commit/6cd24e7d45bfcef2003a581be80882516be64e8d))
* fix release-please permissions by using push trigger ([#50](https://github.com/jsvensson/paletteswap/issues/50)) ([212eb77](https://github.com/jsvensson/paletteswap/commit/212eb7726e7c41cee739c2a7e1f1ccbbc5eaeafb))

## [0.1.2](https://github.com/jsvensson/paletteswap/compare/v0.1.1...v0.1.2) (2026-02-12)


### Bug Fixes

* **lsp:** add DefinitionProvider capability for Zed compatibility ([#46](https://github.com/jsvensson/paletteswap/issues/46)) ([7078b5d](https://github.com/jsvensson/paletteswap/commit/7078b5d3d9a2e258b862ad0c991d9ccc4f73f7c9))


### CI/CD

* remove draft setting to enable automatic releases ([0f01f8f](https://github.com/jsvensson/paletteswap/commit/0f01f8f0c88f35bc8fe1919351fe131af3c7960e))

## [0.1.1](https://github.com/jsvensson/paletteswap/compare/v0.1.0...v0.1.1) (2026-02-10)


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


### Documentation

* add release process documentation ([#36](https://github.com/jsvensson/paletteswap/issues/36)) ([e82abad](https://github.com/jsvensson/paletteswap/commit/e82abad958f98a7c38c9ef4aeb96f8b0db2fb173))


### Code Refactoring

* rename SyntaxStyle to just Style ([a4cd27d](https://github.com/jsvensson/paletteswap/commit/a4cd27dd1f0577f3606b70dfd34d6810cee06a7d))
* restructure packages for cleaner public API ([#16](https://github.com/jsvensson/paletteswap/issues/16)) ([1d6a09e](https://github.com/jsvensson/paletteswap/commit/1d6a09e17c8029cdb53abdc991a69c7e77a5a811))
* update engine and theme to use *color.Node for palette ([4d52bf3](https://github.com/jsvensson/paletteswap/commit/4d52bf3a72c77cbaddc7f5a0ecfdb27314705f66))


### CI/CD

* add release workflow for building and publishing binaries ([#34](https://github.com/jsvensson/paletteswap/issues/34)) ([6c297ec](https://github.com/jsvensson/paletteswap/commit/6c297eca8a3a1e6213f051f17c1c8fb4c0b7e947))
* add workflow to create release PR on manual trigger ([#33](https://github.com/jsvensson/paletteswap/issues/33)) ([c981f14](https://github.com/jsvensson/paletteswap/commit/c981f147344a991a6b8c50065d477d6a6e362f64))
* automated release PR workflow ([#38](https://github.com/jsvensson/paletteswap/issues/38)) ([d83ee32](https://github.com/jsvensson/paletteswap/commit/d83ee3299b6a02bcc3b101e71c56a2ee70098018))
* configure release-please for v0.x releases ([#41](https://github.com/jsvensson/paletteswap/issues/41)) ([ef27095](https://github.com/jsvensson/paletteswap/commit/ef27095afa3e07667b655c4c9c82ea68971c78c9))
* fix squash/rebase merge support ([#39](https://github.com/jsvensson/paletteswap/issues/39)) ([ec0270c](https://github.com/jsvensson/paletteswap/commit/ec0270cf48d1e3f9994b39a084bd1c5352cefcc3))
* remove inline options from workflow ([#42](https://github.com/jsvensson/paletteswap/issues/42)) ([4a8629e](https://github.com/jsvensson/paletteswap/commit/4a8629eb6099cc664a16db40b74ffcda1e978292))
* remove inline options from workflow, use config file only ([4a8629e](https://github.com/jsvensson/paletteswap/commit/4a8629eb6099cc664a16db40b74ffcda1e978292))
* set initial version to 0.1.0 ([#44](https://github.com/jsvensson/paletteswap/issues/44)) ([4b37626](https://github.com/jsvensson/paletteswap/commit/4b37626eaa8e657258d117965098c7c1137a18f2))
