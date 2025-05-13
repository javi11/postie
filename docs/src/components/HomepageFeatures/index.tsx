import type { ReactNode } from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'High Performance',
    Svg: require('@site/static/img/throughput.svg').default,
    description: (
      <>
        Built with Go and optimized for speed, Postie leverages rapidyenc for
        high-performance yEnc encoding and connection pooling for maximum throughput.
      </>
    ),
  },
  {
    title: 'Reliable & Secure',
    Svg: require('@site/static/img/trusted.svg').default,
    description: (
      <>
        Multi-server support with automatic failover, post verification, and
        configurable obfuscation policies to protect your uploads.
      </>
    ),
  },
  {
    title: 'Flexible & Automated',
    Svg: require('@site/static/img/automated.svg').default,
    description: (
      <>
        Configure posting schedules, file watching for automatic uploads,
        PAR2 recovery file generation, and more to meet your specific needs.
      </>
    ),
  },
];

function Feature({ title, Svg, description }: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props) => (
            <Feature key={props.title} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
