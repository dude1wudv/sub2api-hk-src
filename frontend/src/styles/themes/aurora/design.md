# 星澜：海岸天文台

## 艺术方向
情绪冷静、清醒且有远方感。浅色是晨光下的银蓝矿物墙、白色石灰岩建筑和海天水平线；深色独立生成夜海、被侧光勾出的深靛矿物与稀疏星点。空间靠建筑开口与真实光照成立，不使用极光渐变、玻璃球或装饰曲线。视觉重心在右侧，左侧安静承载阅读。

## 界面语言
深钴蓝主操作、消色青辅助信息。卡片使用实色浅瓷/夜蓝，6px控件和10px卡片圆角；选中导航保留细竖尺，表头有克制的字距。文字、表格、固定列和弹窗由各自不透明材质承载，背景不会穿透数据。状态颜色延用公共语义色阶，避免把主题主色误作状态。图表提供十个独立色相。

## 资产
- coastal-observatory-day.webp：1672×941，187400 bytes。
- coastal-observatory-night.webp：1672×941，106454 bytes。
- 内置 image_gen 生成，Sharp WebP quality 82 / effort 6 压缩，未用 CLI API。
- 已查看两张原图：浅色矿物纹理清晰，右侧有自然海岸层次；深色有独立海面月光与矿物侧光，非浅色黑罩。
- 页面背景居中 cover；手机以75%位置裁切保留建筑开口。hero手机向左裁切到安静石面，优先文字可读性。

## 最终生成提示词

### 浅色
Use case: stylized-concept. Asset type: production wallpaper for a dense data management web application, not a UI mockup. Create a serene daylight coastal observatory architecture study, landscape 16:9. Cool silver-blue limestone wall and quiet pale blue atmospheric space occupy the entire left 70 percent with delicate real mineral grain, almost uniform low contrast for reading. Along the rightmost 25 percent a precise monolithic vertical edge opens onto a distant calm blue-grey sea, a thin horizon, layered stone ledge near bottom right, natural optical haze, subtle tactile mineral texture. Architectural editorial photography, exceptionally restrained and refined, cool morning diffuse skylight, desaturated chalk white and steel blue, no purple. The composition must work cropped on desktop and mobile. Main visual interest confined to right edge and lower right, not the center. No text, lettering, UI, buttons, stars icon, symbols, logos, furniture, people, gradients as graphic decoration, curving ribbons, glass blobs, shiny chrome or neon. Actual believable material and soft light, no heavy shadows. Generate a clean high quality wide background image.

### 深色
Use case: stylized-concept. Asset type: production dark-mode background wallpaper for dense data management application, no interface. A nocturnal coastal observatory in architectural editorial photography, wide 16:9 landscape. The left 70 percent is a quiet deep blue-grey mineral plaster plane with delicate matte grain, softly illuminated by very faint cool reflected sky light; dark but visibly tactile, low contrast reading space. Rightmost 25 percent opens beyond a crisp stone vertical edge to midnight ocean and indigo sky, a precise far horizon with three tiny faint distant stars, a moonlit slate ledge low on the right. Independent nighttime lighting, credible spatial depth and quiet high-end architectural mood. Restrained desaturated deep navy, slate, a hint of silver blue. No bright moon, no glowing nebula, no purple, no neon, no colorful aurora, no decorative curves, no glass, no glossy plastic, no text, no letters, no symbols, no UI, no buttons, no logos or people. Fine restrained detail at right edge, center and left calm. Mobile crop must remain attractive. Real material and grazing light, never a flat gradient. Wide high quality background.

## 验证边界
主 Agent 统一集成与页面截图验收。子 Agent 不启动共享浏览器、不提交、不部署。
