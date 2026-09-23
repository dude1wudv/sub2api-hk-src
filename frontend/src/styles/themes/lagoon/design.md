# 杏纱 · 装帧工坊

设计方向：将「纱」解释为能触摸到的棉麻经纬，将「杏」解释为纸张与植物染料的温度。整体是安静、亲近而严谨的装帧工作室。浅色以奶油棉纸为主要材质，漫射晨光照亮右侧杏色亚麻；深色以葡萄褐书布为主要材质，右侧织物接住温暖灯光，文字表面使用深梅褐色。两个场景独立生成，不叠黑色遮罩。

## 页面语言

页面主色是陶土，梅紫为次色；语义绿、蓝、橙、红继续承担原有数据意义。界面是一层层纸页：不透明内容卡、单像素纸页边界、窄书脊导航与少量双细线。12px 卡片、7px 控件、14px 弹窗圆角，拒绝药丸按钮和高光玻璃。宽松的标题与细密的表格形成节奏，数字保留等宽对齐。背景只在框架间隙及轻透框架中出现，卡片、输入、浮层和表格保持清楚。

表格表头使用略深纸色，正文及固定列共用不透明表面；悬停行使用杏色/梅褐色实底。键盘焦点为陶土/浅杏色。图表提供十种独立色，避免整张图只使用同一主色深浅。

## 资产

内置 image_gen 工具生成。原图均为 1536 × 1024；使用 Pillow 仅按用户要求压缩为 WebP，quality=82、method=6，未修改内容。

- `atelier-day.webp`：167196 bytes，奶油棉纸与杏色亚麻。
- `atelier-night.webp`：265328 bytes，夜间书布与灯光下的亚麻。

已分别查看两张生成图，并复查压缩夜景，纤维与织边仍可辨认，没有文字、UI、伪影性高光。桌面居中 cover，手机 77% 水平位置取景，在窄幅边缘留出织物层次；图像中心低对比，主要阅读区域由实体纸面覆盖。实际页面截图与浏览器验收由主 Agent 整合后完成，不把素材检查当作页面验收。

## 最终生成提示词

### 浅色

Use case: photorealistic-natural. Asset type: production desktop application background wallpaper, pure material photograph, wide landscape 3:2. Create an exquisite quiet editorial still life of a bookbinding textile atelier in diffuse morning daylight. View straight down at broad warm ivory cotton rag paper with extremely fine irregular fibers; a wide muted pale apricot linen swatch folds gently along far right edge and lower-right corner, a thin handmade paper edge is visible near upper-left edge. Material details tactile and convincingly real, tiny woven threads, softly irregular paper edges, natural contact shadows. 75 percent center and left area must remain calm nearly uniform cream paper with only very faint fiber texture; all stronger details are confined to outermost 20 percent especially right. Composition spacious, precise, cultured, minimal but never digitally flat. Light is broad diffuse light from upper left, no dramatic spotlight. Palette warm off-white, chalk, pale dried apricot, subtle dusty rose textile edge. Must work cropped to narrow mobile center. This is ONLY background material imagery, not UI, not mockup. No text, no words, no objects such as books pens flowers, no buttons, no tables, no frames, no glass, no curved graphic decorations, no illustration, no synthetic gradient.

### 深色

Use case: photorealistic-natural. Asset type: production application dark-mode background wallpaper, pure material photograph, wide landscape 3:2. A night-time bookbinding and textile atelier, viewed directly top-down: broad warm dark aubergine-brown cotton bookcloth with extremely fine visible weave occupies 80 percent of composition, calm, even, matte. At the far right edge and lower-right corner, a loose length of washed dusty-rose linen with a visible sewn selvage is gently folded, catching a narrow but soft pool of apricot lamplight from outside frame to right. A thin layered edge of dark plum paper at far upper-left, understated contact shadow. The center and left are quietly lit and evenly dark, low contrast fine material texture, not pure black. The right edge textile has clearly separate planes of warm light, midtone and subtle shadow; beautiful real woven fibers, restrained editorial photography, intimate quiet evening mood. Colors: dark raisin/plum brown base, faded rose fabric, delicate warm amber/apricot lamplight without metallic shine. Independent night scene, not daytime with black overlay. Cropping narrow center must remain beautiful and calm. No UI, no text, no logos, no books pens flowers or other objects, no glass, no graphic curves, no digital gradient, no bright center, no glossy silk, no glare.

## 页面截图迭代

查看主 Agent 提供的浅/深仪表盘截图后，取消与星澜相似的 hero 左侧主色竖线。改为书籍扉页构图：上下双细规线，宋/明朝体标题，右侧今日请求以单根竖分隔线融入纸页而非独立卡盒。主框架左侧近直角、右侧保留柔圆角，像装订的一张书页。数据卡仍使用无花纹的实底，保证密集数据阅读。手机标题降至 25px，保留扉页节奏。

静态对比度核对：浅色次要文字 5.75:1，深色次要文字 6.10:1，主按钮文字 5.94:1，浅/深主色文字分别 7.44:1、8.13:1。
