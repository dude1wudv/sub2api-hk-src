# 墨铜：精密档案工坊

## 艺术方向
情绪克制、安定、有秩序。浅色是粉白矿石与旧铜嵌条，冷白日光揭示细粒石面；深色是碳黑烧结石与暗铜板，侧向掠光揭示板材厚度。视觉重心置于右缘，中央保留安静阅读空间。两张图独立生成，夜景不是对白昼图叠黑。

实体面板、4–7px 小圆角、细索引线和几乎无阴影的层次体现精密感。铜色指示主要操作及导航选择，氧化青用于辅助数据。灰阶带极轻矿物绿调，避免整页棕黄。信息面板和固定列使用实心填充，图片只通过框架边缘与间隙显露。语义成功/警告/错误延续公共绿色/琥珀/红色区分。

## 资产与实现
- 内置 image_gen，分别生成并查看两张原始图，再压缩并再次查看 WebP。
- mineral-day.webp：1536×1024，273130 bytes；mineral-night.webp：1536×1024，139116 bytes。WebP quality 78 / method 6。
- 桌面右对齐 cover，手机 88% center 保留右边铜嵌条。两张图不含 UI、文字、按钮或表格。
- graphite.css 提供完整浅深色 tokens、独立 chart 10 色、表格正常/悬停/固定列色、菜单/弹窗材质与焦点，并使用公共 material-system.css 组件契约。
- 不更改业务组件、权限、字段或操作入口。未启动浏览器或运行全套测试；实际页面验收由主 Agent 集中完成。

## 最终生图 Prompt — Day
Use case: stylized-concept. Asset type: production website background texture, wide 3:2 landscape, no UI.
Create a restrained architectural material study for a precise archive workshop: an immense pale ivory-white honed limestone / mineral plaster slab fills 80 percent of the composition with extremely subtle physically realistic fine grain. At the far right edge (last 14 percent) a narrow vertically inset aged copper plate, desaturated burnt orange patina, finely brushed metal and a slim dark recessed seam. Bottom right a small stepped overlap of limestone slabs and metal, very understated. Main middle 75 percent is nearly uniform quiet off-white negative space. Cool diffuse daylight from upper left gives a gently tangible plane, delicate material texture, no dramatic shadows. Premium architectural photography, real material depth, restrained contrast, crisp not blurred. Edges remain interesting when cropped on phone. No text, letters, logos, objects, furniture, interface, buttons, tables, gradients, waves, curves, glowing streaks, glass or marble veins. Overall very light, calm, rigorous.

## 最终生图 Prompt — Night
Use case: stylized-concept. Asset type: production website background texture, wide 3:2 landscape, no UI.
A precise architectural material composition in a dark archive workshop. Matte fine-grained graphite-black sintered stone fills the quiet central and left 80 percent, subtle tiny mineral pores, smooth honed surface. Along the extreme right edge a narrow recessed panel of aged dark copper metal with brushed horizontal striation and restrained burgundy-brown patina. Bottom right two crisp overlapping black stone slabs create subtle physical spatial depth. Warm low grazing illumination from the right reveals only the copper edge and slab thickness, cool very soft ambient bounce keeps the charcoal central plane readable. Not a bright daytime image darkened: purpose-built nocturnal architectural lighting with black materials and localized light. Main 75 percent negative space is nearly uniform neutral charcoal. Premium material photography, clear exact edges, restrained detail and contrast. No text, letters, logos, furniture, UI, buttons, tables, gradients, waves, curves, glowing lines, sparkles, glass, blur, vignettes or marble veins. Mobile crop should still show physical material.


## 静态验证
PostCSS 成功解析样式。关键颜色按 WCAG 相对亮度公式计算：浅色正文 14.27:1、次要文字 5.52:1、选中态 7.58:1；白字铜色按钮 7.02:1；深色正文 13.85:1、次要文字 7.32:1、选中态 8.04:1。此处仅为 token 对比度，不能替代真实页面验收。
