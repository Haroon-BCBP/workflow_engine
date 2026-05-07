export default class CustomContextPad {
  static readonly $inject = [
    'config.contextPad',
    'contextPad',
    'create',
    'elementFactory',
    'injector',
    'translate'
  ];

  private readonly create: any;
  private readonly elementFactory: any;
  private readonly injector: any;
  private readonly translate: any;

  constructor(
    _config: any,
    contextPad: any,
    create: any,
    elementFactory: any,
    injector: any,
    translate: any
  ) {
    this.create = create;
    this.elementFactory = elementFactory;
    this.injector = injector;
    this.translate = translate;

    contextPad.registerProvider(this);
  }

  getContextPadEntries(element: any) {
    const {
      create,
      elementFactory,
      translate
    } = this;

    const actions: any = {};

    if (element.type === 'label') {
      return actions;
    }

    function appendAction(type: string, className: string, title: string, options?: any) {
      function appendListener(event: any, element: any) {
        const shape = elementFactory.createShape({ type, ...options });
        create.start(event, shape, element);
      }

      return {
        group: 'model',
        className,
        title: translate(title),
        action: {
          dragstart: appendListener,
          click: appendListener
        }
      };
    }

    // Allowed connections/appends
    if (element.type !== 'bpmn:EndEvent' && element.type !== 'bpmn:Participant' && element.type !== 'bpmn:Lane') {
      actions['append.end-event'] = appendAction(
        'bpmn:EndEvent', 'bpmn-icon-end-event-none', 'Append EndEvent'
      );
      actions['append.exclusive-gateway'] = appendAction(
        'bpmn:ExclusiveGateway', 'bpmn-icon-gateway-xor', 'Append Exclusive Gateway'
      );
      actions['append.parallel-gateway'] = appendAction(
        'bpmn:ParallelGateway', 'bpmn-icon-gateway-parallel', 'Append Parallel Gateway'
      );
      actions['append.task'] = appendAction(
        'bpmn:UserTask', 'bpmn-icon-user-task', 'Append Stage (User Task)'
      );
    }

    // Delete is always allowed
    actions['delete'] = {
      group: 'edit',
      className: 'bpmn-icon-trash',
      title: translate('Remove'),
      action: {
        click: (_event: any, element: any) => {
          this.injector.get('modeling').removeElements([element]);
        }
      }
    };

    return actions;
  }
}
