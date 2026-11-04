export type SeriesType = "Manhwa" | "Manhua" | "Manga";

export type Chapter = {
  number: number;
  ago: string;
  isNew?: boolean;
};

export type Series = {
  slug: string;
  title: string;
  tagline: string;
  synopsis: string;
  type: SeriesType;
  genres: string[];
  rating: number;
  chapters: number;
  cover: string;
  views: string;
};

export type Update = {
  slug: string;
  chapters: Chapter[];
};

export type Period = "daily" | "weekly" | "monthly";

// shortcut: dados estáticos do design base; trocar por fetch(`${API}/v1/series`)
// com next.tags quando a API Go (api/openapi.yaml) expor os endpoints reais.
export const series: Series[] = [
  {
    slug: "solo-leveling",
    title: "Solo Leveling",
    tagline: "O caçador mais fraco se torna o mais forte",
    synopsis:
      "Em um mundo onde portais conectam dimensões e caçadores combatem monstros, Sung Jinwoo, o caçador de rank E mais fraco, recebe um poder misterioso que o faz subir de nível sem limites.",
    type: "Manhwa",
    genres: ["Ação", "Fantasia"],
    rating: 4.9,
    chapters: 179,
    cover: "/covers/solo-leveling.png",
    views: "12.4K",
  },
  {
    slug: "tower-of-god",
    title: "Tower of God",
    tagline: "Suba a torre e conquiste tudo",
    synopsis:
      "Twenty Fifth Bam entra na Torre para perseguir sua amiga Rachel, e descobre que cada andar guarda provas, aliados e inimigos.",
    type: "Manhwa",
    genres: ["Fantasia", "Aventura"],
    rating: 4.8,
    chapters: 550,
    cover: "/covers/tower-of-god.png",
    views: "9.8K",
  },
  {
    slug: "the-beginning-after-the-end",
    title: "The Beginning After The End",
    tagline: "Uma segunda vida em um mundo de magia",
    synopsis:
      "Reincarnado como o bebê Reytheon em um mundo de espadas e magia, o antigo rei mantém suas memórias e cresce para proteger sua nova família.",
    type: "Manhwa",
    genres: ["Fantasia", "Ação"],
    rating: 4.9,
    chapters: 200,
    cover: "/covers/beginning-after-the-end.png",
    views: "7.6K",
  },
  {
    slug: "omniscient-reader",
    title: "Omniscient Reader's Viewpoint",
    tagline: "O único leitor vive o último capítulo",
    synopsis:
      "Kim Dokja é o único que terminou um web novel obscuro — quando o mundo se torna a história, ele conhece o final de cada cenário.",
    type: "Manhwa",
    genres: ["Aventura", "Fantasia"],
    rating: 4.8,
    chapters: 178,
    cover: "/covers/omniscient-reader.png",
    views: "8.2K",
  },
  {
    slug: "return-of-the-mount-hua-sect",
    title: "Return of the Mount Hua Sect",
    tagline: "O Sábio Celestial volta para reconstruir sua seita",
    synopsis:
      "Dependendo de sua morte, Chung Myung desperta mil anos depois e encontra a Seita de Mount Hua em ruínas. Hora de colocar os discípulos para trabalhar.",
    type: "Manhwa",
    genres: ["Ação", "Comédia", "Artes Marciais"],
    rating: 4.7,
    chapters: 130,
    cover: "/covers/mount-hua-sect.png",
    views: "5.9K",
  },
  {
    slug: "windbreaker",
    title: "Windbreaker",
    tagline: "O vento muda de direção",
    synopsis:
      "Acima de todos está Shin Jae, um estudante modelo com segredos. Queda de status, renascimento como delinquente e o vento que carrega novos encontros.",
    type: "Manhwa",
    genres: ["Ação", "Comédia", "Escolar"],
    rating: 4.6,
    chapters: 480,
    cover: "/covers/windbreaker.png",
    views: "5.2K",
  },
  {
    slug: "nano-machine",
    title: "Nano Machine",
    tagline: "Tecnologia do futuro nas artes marciais",
    synopsis:
      "Herdeiro desprezado do Culto Demoníaco, Cheon Woon recebe nano-máquinas de um descendente distante e passa de caçado a caçador.",
    type: "Manhwa",
    genres: ["Sci-fi", "Ação"],
    rating: 4.5,
    chapters: 165,
    cover: "/covers/nano-machine.png",
    views: "7.1K",
  },
  {
    slug: "shadow-monarch",
    title: "Shadow Monarch",
    tagline: "O monarca comanda as sombras",
    synopsis:
      "Depois de sobreviver a um massacre, Yeon Woo desperta com o poder de erguer sombras guerreiras — e o mundo de portais nunca mais será o mesmo.",
    type: "Manhwa",
    genres: ["Dark Fantasy", "Ação"],
    rating: 4.7,
    chapters: 98,
    cover: "/covers/shadow-monarch.png",
    views: "6.5K",
  },
  {
    slug: "martial-peak",
    title: "Martial Peak",
    tagline: "Do portador de água ao pico das artes marciais",
    synopsis:
      "Yang Kai, um simples coletante de ervas, encontra uma técnica esquecida e inicia a longa escalada do cultivo até o ápice marcial.",
    type: "Manhua",
    genres: ["Artes Marciais", "Cultivo"],
    rating: 4.5,
    chapters: 320,
    cover: "/covers/martial-peak.png",
    views: "4.8K",
  },
  {
    slug: "portal-hunter",
    title: "Portal Hunter",
    tagline: "Um caçador, infinitos portais",
    synopsis:
      "Han Seojun sobrevive ao primeiro portal e descobre que pode entrar neles quando quer — o que atrai atenção que nenhum caçador quer.",
    type: "Manhwa",
    genres: ["Ação", "Aventura", "Sci-fi"],
    rating: 4.4,
    chapters: 87,
    cover: "/covers/portal-hunter.png",
    views: "3.9K",
  },
];

export const bySlug = (slug: string): Series | undefined =>
  series.find((s) => s.slug === slug);

export const featured = ["solo-leveling", "tower-of-god", "the-beginning-after-the-end"].map(
  (slug) => bySlug(slug)!
);

// ordem do carrossel Top 10 no design base
export const top10 = [
  "solo-leveling",
  "tower-of-god",
  "the-beginning-after-the-end",
  "omniscient-reader",
  "return-of-the-mount-hua-sect",
  "windbreaker",
  "nano-machine",
  "shadow-monarch",
  "martial-peak",
  "portal-hunter",
].map((slug, i) => ({ rank: i + 1, series: bySlug(slug)! }));

export const updates: Update[] = [
  { slug: "solo-leveling", chapters: [{ number: 179, ago: "1 hora atrás", isNew: true }, { number: 178, ago: "3 horas atrás" }] },
  { slug: "tower-of-god", chapters: [{ number: 550, ago: "2 horas atrás", isNew: true }, { number: 549, ago: "1 dia atrás" }] },
  { slug: "the-beginning-after-the-end", chapters: [{ number: 200, ago: "4 horas atrás", isNew: true }, { number: 199, ago: "1 dia atrás" }] },
  { slug: "omniscient-reader", chapters: [{ number: 178, ago: "30 min atrás", isNew: true }, { number: 177, ago: "5 horas atrás" }] },
  { slug: "return-of-the-mount-hua-sect", chapters: [{ number: 130, ago: "6 horas atrás", isNew: true }, { number: 129, ago: "2 dias atrás" }] },
  { slug: "windbreaker", chapters: [{ number: 480, ago: "12 horas atrás", isNew: true }, { number: 479, ago: "3 dias atrás" }] },
  { slug: "nano-machine", chapters: [{ number: 165, ago: "8 horas atrás", isNew: true }, { number: 164, ago: "2 dias atrás" }] },
  { slug: "shadow-monarch", chapters: [{ number: 98, ago: "1 dia atrás", isNew: true }, { number: 97, ago: "4 dias atrás" }] },
  { slug: "martial-peak", chapters: [{ number: 320, ago: "1 dia atrás", isNew: true }, { number: 319, ago: "3 dias atrás" }] },
  { slug: "portal-hunter", chapters: [{ number: 87, ago: "5 horas atrás", isNew: true }, { number: 86, ago: "2 dias atrás" }] },
];

const pick = (slugs: string[]) => slugs.map((slug) => bySlug(slug)!);

export const popular: Record<Period, Series[]> = {
  daily: pick(["solo-leveling", "tower-of-god", "omniscient-reader", "nano-machine", "shadow-monarch"]),
  weekly: pick(["tower-of-god", "solo-leveling", "the-beginning-after-the-end", "omniscient-reader", "return-of-the-mount-hua-sect"]),
  monthly: pick(["the-beginning-after-the-end", "solo-leveling", "tower-of-god", "windbreaker", "shadow-monarch"]),
};

export const ranking = pick(top10.map((t) => t.series.slug));
