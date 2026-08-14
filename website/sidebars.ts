import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {type: 'doc', id: 'intro', label: 'Overview'},
    {
      type: 'category',
      label: 'Data Model',
      collapsed: false,
      items: [
        'data-model/mcd',
        'data-model/mld',
        'data-model/mpd',
      ],
    },
  ],
};

export default sidebars;
