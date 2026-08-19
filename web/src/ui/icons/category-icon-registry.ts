import {
  BookOpen,
  BriefcaseBusiness,
  Cloud,
  Code,
  Database,
  Film,
  Folder,
  Gamepad2,
  Globe,
  GraduationCap,
  HeartPulse,
  House,
  Music,
  Palette,
  ShoppingBag,
  Star,
  Wrench,
  type LucideIcon,
} from 'lucide-preact';

export const categoryIconRegistry = {
  folder: { component: Folder, label: '文件夹', keywords: ['默认', '目录'] },
  house: { component: House, label: '主页', keywords: ['首页', '家庭'] },
  globe: { component: Globe, label: '网络', keywords: ['网站', '全球'] },
  code: { component: Code, label: '开发', keywords: ['编程', '代码'] },
  database: {
    component: Database,
    label: '数据',
    keywords: ['数据库', '存储'],
  },
  briefcase: {
    component: BriefcaseBusiness,
    label: '工作',
    keywords: ['办公', '职业'],
  },
  book: { component: BookOpen, label: '阅读', keywords: ['书籍', '知识'] },
  graduation: {
    component: GraduationCap,
    label: '学习',
    keywords: ['教育', '课程'],
  },
  palette: { component: Palette, label: '设计', keywords: ['创意', '颜色'] },
  film: { component: Film, label: '影视', keywords: ['电影', '视频'] },
  music: { component: Music, label: '音乐', keywords: ['音频', '歌曲'] },
  game: { component: Gamepad2, label: '游戏', keywords: ['娱乐', '主机'] },
  shopping: {
    component: ShoppingBag,
    label: '购物',
    keywords: ['商店', '消费'],
  },
  health: { component: HeartPulse, label: '健康', keywords: ['医疗', '运动'] },
  cloud: { component: Cloud, label: '云服务', keywords: ['云端', '托管'] },
  star: { component: Star, label: '收藏', keywords: ['常用', '精选'] },
  tools: { component: Wrench, label: '工具', keywords: ['实用', '维护'] },
} satisfies Record<
  string,
  { component: LucideIcon; label: string; keywords: readonly string[] }
>;

export type CategoryIconName = keyof typeof categoryIconRegistry;

export function findCategoryIcons(query: string) {
  const normalized = query.trim().toLocaleLowerCase('zh-CN');
  const entries = Object.entries(categoryIconRegistry) as Array<
    [CategoryIconName, (typeof categoryIconRegistry)[CategoryIconName]]
  >;

  if (!normalized) {
    return entries;
  }

  return entries.filter(([name, item]) =>
    [name, item.label, ...item.keywords].some((value) =>
      value.toLocaleLowerCase('zh-CN').includes(normalized),
    ),
  );
}
