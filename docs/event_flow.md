# Event Flow Diagram

```ascii
          DockerHub Event Provider        Hermes (trigger-manager)       pipeline manager (cfapi)

                    +                            +                             +
original event      |                            |                             |
                    |                            |                             |
        +---------> |  normalized event (vars)   |                             |
                    |                            |                             |
                    | +------------------------> |     pipeline: p1(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |     OK: running p1,p2,p3   |                             |
                    | <------------------------+ |                             |
                    |                            |     pipeline: p2(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |                            |                             |
                    |                            |                             |
                    |                            |     pipeline: p3(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |                            |                             |
                    |                            |                             |
                    +                            +                             +

```